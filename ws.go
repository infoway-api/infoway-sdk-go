package infoway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultWSURL            = "wss://data.infoway.io/ws"
	heartbeatInterval       = 30 * time.Second
	initialBackoff          = time.Second
	maxWSBackoff            = 30 * time.Second
	defaultStaleAfter       = 90 * time.Second
	defaultSubscribeRetry   = 2 * time.Second
	defaultAckTimeout       = 8 * time.Second
	subscribeFailPrefix     = "Subscribe fail:"
	frameBudget             = 60
	// Two heartbeats per minute, plus room for subscribe/unsubscribe frames.
	heartbeatReserve        = 8
	outboundStampLimit      = 256
	// A frame aged exactly one window is still on the closed edge. Step past it.
	windowSlack             = time.Millisecond
)

// WSOptions configures a quotes WebSocket.
type WSOptions struct {
	APIKey               string
	Business             Business
	BaseURL              string
	MaxReconnectAttempts int // 0 = unlimited
	PrintFrames          bool
	HeartbeatInterval    time.Duration
	ReconnectBackoff     time.Duration
	// StaleAfter resubscribes when this socket has seen ticks then gone silent.
	// Default 90s. 0 disables the watchdog.
	StaleAfter time.Duration
	// SubscribeRetry is the first delay after a 516. Default 2s.
	SubscribeRetry time.Duration
}

type subscription struct {
	kind      string
	codes     string
	klineType int
	includeTy bool
}

// WebSocket is the market-data socket (wss://data.infoway.io/ws?business=…).
type WebSocket struct {
	url          string
	maxReconnect int
	printFrames  bool
	hbInterval      time.Duration
	initBackoff     time.Duration
	staleAfter      time.Duration
	subscribeRetry  time.Duration
	ackTimeout      time.Duration
	hbGens          int
	OnTrade      func(map[string]any)
	OnDepth      func(map[string]any)
	OnKline      func(map[string]any)
	OnFrame      func(string)
	OnError      func(error)
	OnReconnect  func()
	OnDisconnect func()

	mu                   sync.Mutex
	subs                 map[string]subscription
	conn                 *websocket.Conn
	stop                 chan struct{}
	running              bool
	connected            bool
	hadPush              bool
	lastPush             time.Time
	lastSubscribe        time.Time
	lastResub            time.Time
	pendingAcks          int
	outbound             []time.Time
	staleRetries         int
	subscribeFailRetries int
	retryTimer           *time.Timer
}

// NewWebSocket requires APIKey (or INFOWAY_API_KEY) and Business.
func NewWebSocket(opts WSOptions) (*WebSocket, error) {
	key, err := requireAPIKey(opts.APIKey)
	if err != nil {
		return nil, err
	}
	if opts.Business == "" {
		return nil, fmt.Errorf("business is required (stock/japan/india/korea/taiwan/crypto/common)")
	}
	base := opts.BaseURL
	if base == "" {
		base = defaultWSURL
	}
	hb := opts.HeartbeatInterval
	if hb <= 0 {
		hb = heartbeatInterval
	}
	bo := opts.ReconnectBackoff
	if bo <= 0 {
		bo = initialBackoff
	}
	stale := defaultStaleAfter
	if opts.StaleAfter > 0 {
		stale = opts.StaleAfter
	}
	retry := defaultSubscribeRetry
	if opts.SubscribeRetry > 0 {
		retry = opts.SubscribeRetry
	}
	return &WebSocket{
		url:            fmt.Sprintf("%s?business=%s&apikey=%s", base, opts.Business, key),
		maxReconnect:   opts.MaxReconnectAttempts,
		printFrames:    opts.PrintFrames,
		hbInterval:     hb,
		initBackoff:    bo,
		staleAfter:     stale,
		subscribeRetry: retry,
		ackTimeout:     defaultAckTimeout,
		subs:           map[string]subscription{},
	}, nil
}

// Connect runs the read loop until ctx is cancelled, Close is called, or a 401 handshake.
func (w *WebSocket) Connect(ctx context.Context) error {
	w.mu.Lock()
	w.running = true
	stop := make(chan struct{})
	w.stop = stop
	w.mu.Unlock()

	backoff := w.initBackoff
	if backoff <= 0 {
		backoff = initialBackoff
	}
	attempt := 0
	for {
		w.mu.Lock()
		running := w.running
		w.mu.Unlock()
		if !running {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stop:
			return nil
		default:
		}

		opened, err := w.connectOnce(ctx)
		if opened {
			backoff = w.initBackoff
			attempt = 0
		}
		w.mu.Lock()
		running = w.running
		w.mu.Unlock()
		if !running {
			return nil
		}
		if err != nil {
			if _, ok := err.(*AuthError); ok {
				w.emitError(err)
				return err
			}
			w.emitError(err)
		}
		attempt++
		if w.maxReconnect > 0 && attempt > w.maxReconnect {
			return err
		}
		w.safeRun(w.OnDisconnect)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-stop:
			timer.Stop()
			return nil
		case <-timer.C:
		}
		if backoff < maxWSBackoff {
			backoff *= 2
			if backoff > maxWSBackoff {
				backoff = maxWSBackoff
			}
		}
	}
}

func (w *WebSocket) connectOnce(ctx context.Context) (opened bool, err error) {
	w.mu.Lock()
	leftover := w.conn
	w.conn = nil
	w.hadPush = false
	w.lastPush = time.Time{}
	w.pendingAcks = 0
	w.stopRetryLocked()
	w.mu.Unlock()
	if leftover != nil {
		_ = leftover.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "reconnect"), time.Now().Add(2*time.Second))
		_ = leftover.Close()
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		ReadBufferSize:   16 * 1024,
		WriteBufferSize:  4 * 1024,
	}
	conn, resp, err := dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return false, newAuthError(wsAuthMessage(""), "")
		}
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			return false, OfHTTPStatus(resp.StatusCode, fmt.Sprintf("WebSocket handshake failed with HTTP %d", resp.StatusCode), "")
		}
		return false, err
	}

	w.mu.Lock()
	reconnected := w.connected
	w.conn = conn
	w.connected = true
	w.resubscribeLocked(true)
	w.mu.Unlock()
	if reconnected {
		w.safeRun(w.OnReconnect)
	}

	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	w.hbGens++
	go w.heartbeat(hbCtx, conn)
	go w.watchdog(hbCtx, conn)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			w.mu.Lock()
			if w.conn == conn {
				w.conn = nil
			}
			w.stopRetryLocked()
			w.mu.Unlock()
			cancel()
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "reconnect"), time.Now().Add(time.Second))
			_ = conn.Close()
			return true, err
		}
		w.dispatch(raw)
	}
}

func (w *WebSocket) heartbeat(ctx context.Context, conn *websocket.Conn) {
	iv := w.hbInterval
	if iv <= 0 {
		iv = heartbeatInterval
	}
	t := time.NewTicker(iv)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.mu.Lock()
			if w.conn != conn {
				w.mu.Unlock()
				return
			}
			err := w.sendLocked(map[string]any{"code": int(WsHeartbeat), "trace": traceID()})
			w.mu.Unlock()
			if err != nil {
				_ = conn.Close()
				return
			}
		}
	}
}

func parseWSFrame(raw []byte) (map[string]any, bool) {
	body := strings.TrimSpace(string(raw))
	if strings.HasPrefix(body, subscribeFailPrefix) {
		rest := strings.TrimSpace(body[len(subscribeFailPrefix):])
		if rest != "" && !strings.Contains(rest, "{") {
			return map[string]any{"msg": rest, "subscribeFailProse": true}, true
		}
		body = rest
	}
	if i := strings.IndexByte(body, '{'); i > 0 {
		body = body[i:]
	}
	if body == "" || body[0] != '{' {
		return nil, false
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		return nil, false
	}
	return msg, true
}

// subscribeFailProse is the text after "Subscribe fail:" when the server sent no JSON.
func subscribeFailProse(raw []byte) (string, bool) {
	body := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(body, subscribeFailPrefix) {
		return "", false
	}
	rest := strings.TrimSpace(body[len(subscribeFailPrefix):])
	if rest == "" || strings.Contains(rest, "{") {
		return "", false
	}
	return rest, true
}

// quotaRetryDelay is how long a 516 resubscribe waits. The server counts every
// inbound frame, including heartbeats, toward 60 per minute. heartbeatReserve
// frames stay unused so heartbeats and a few application frames still fit.
func quotaRetryDelay(backoff time.Duration, groups int, sent []time.Time, now time.Time, budget int, window time.Duration) time.Duration {
	n := groups
	if n < 1 {
		n = 1
	}
	room := budget - heartbeatReserve
	recent := make([]time.Time, 0, len(sent))
	for _, t := range sent {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].Before(recent[j]) })
	if n > room {
		if backoff > window {
			return backoff
		}
		return window
	}
	free := room - len(recent)
	if free >= n {
		return backoff
	}
	need := n - free
	if need > len(recent) {
		if backoff > window {
			return backoff
		}
		return window
	}
	expire := window - now.Sub(recent[need-1])
	if backoff > expire {
		return backoff
	}
	return expire
}

// quotaRetryRemaining is the extra wait when a scheduled 516 retry comes due.
// The backoff was already slept. Returns 0 when the replay fits, and when it
// can never fit in one window. A frame aged exactly one window is still inside
// a closed window, so a positive result is one millisecond past that edge.
func quotaRetryRemaining(groups int, sent []time.Time, now time.Time, budget int, window time.Duration) time.Duration {
	n := groups
	if n < 1 {
		n = 1
	}
	room := budget - heartbeatReserve
	if n > room {
		return 0
	}
	recent := make([]time.Time, 0, len(sent))
	for _, t := range sent {
		if now.Sub(t) <= window {
			recent = append(recent, t)
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].Before(recent[j]) })
	free := room - len(recent)
	if free >= n {
		return 0
	}
	need := n - free
	if need > len(recent) {
		return 0
	}
	expire := window - now.Sub(recent[need-1])
	if expire < 0 {
		expire = 0
	}
	return expire + windowSlack
}

// watchdogResendGap is how long to wait before replaying unacked subscriptions.
// Ten groups or fewer keep ackTimeout. A larger set, replayed on every 15s poll,
// would use up the 60 frames/minute budget by itself.
func watchdogResendGap(groups int, ackTimeout time.Duration) time.Duration {
	n := groups
	if n < 1 {
		n = 1
	}
	if n <= 10 {
		return ackTimeout
	}
	gap := 60 * time.Second * time.Duration(n) / 40
	if gap < 60*time.Second {
		return 60 * time.Second
	}
	return gap
}

func (w *WebSocket) dispatch(raw []byte) {
	if w.printFrames {
		log.Print(string(raw))
	}
	if w.OnFrame != nil {
		w.safeCallStr(w.OnFrame, string(raw))
	}
	msg, ok := parseWSFrame(raw)
	if !ok {
		return
	}
	if prose, isProse := msg["subscribeFailProse"].(bool); isProse && prose {
		w.mu.Lock()
		w.pendingAcks = 0
		if w.retryTimer != nil {
			w.retryTimer.Stop()
			w.retryTimer = nil
		}
		w.mu.Unlock()
		text, _ := msg["msg"].(string)
		w.emitError(&APIError{Ret: 0, Msg: text, ErrorName: "SUBSCRIBE_FAIL"})
		return
	}
	code, ok := asInt(msg["code"])
	if !ok {
		return
	}
	if code == int(WsConnectOK) || code == int(WsHeartApply) {
		return
	}
	if IsWsError(code) {
		trace := strField(msg, "traceId")
		if trace == "" {
			trace = strField(msg, "trace")
		}
		w.emitError(wsFailure(code, strField(msg, "msg"), trace))
		if IsTerminalWs(code) {
			w.stopForTerminal()
			return
		}
		if IsSubscribeQuotaExceeded(code) {
			w.mu.Lock()
			w.scheduleSubscribeRetryLocked()
			w.mu.Unlock()
		} else if code == int(WsErrProductsQuantityExceed) {
			w.mu.Lock()
			w.pendingAcks = 0
			if w.retryTimer != nil {
				w.retryTimer.Stop()
				w.retryTimer = nil
			}
			w.mu.Unlock()
		}
		return
	}
	data, _ := msg["data"].(map[string]any)
	switch WsCode(code) {
	case WsPushTrade:
		w.notePush()
		if data != nil && w.OnTrade != nil {
			w.safeCall(w.OnTrade, data)
		}
	case WsPushDepth:
		w.notePush()
		if data != nil && w.OnDepth != nil {
			w.safeCall(w.OnDepth, data)
		}
	case WsPushKline:
		w.notePush()
		if data != nil && w.OnKline != nil {
			w.safeCall(w.OnKline, data)
		}
	case WsSubTradeAck, WsSubDepthAck, WsSubKlineAck:
		w.noteSubscribeAck()
	}
}

func (w *WebSocket) notePush() {
	w.mu.Lock()
	w.lastPush = time.Now()
	w.hadPush = true
	w.staleRetries = 0
	w.subscribeFailRetries = 0
	w.mu.Unlock()
}

func (w *WebSocket) noteSubscribeAck() {
	w.mu.Lock()
	if w.pendingAcks > 0 {
		w.pendingAcks--
	}
	w.subscribeFailRetries = 0
	w.mu.Unlock()
}

func (w *WebSocket) noteSubscribeSentLocked() {
	w.pendingAcks++
	w.lastSubscribe = time.Now()
}

func (w *WebSocket) stopRetryLocked() {
	if w.retryTimer != nil {
		w.retryTimer.Stop()
		w.retryTimer = nil
	}
}

func (w *WebSocket) scheduleSubscribeRetryLocked() {
	if len(w.subs) == 0 || w.retryTimer != nil {
		return
	}
	w.pendingAcks = 0
	shift := w.subscribeFailRetries
	if shift > 4 {
		shift = 4
	}
	backoff := w.subscribeRetry * time.Duration(1<<shift)
	if backoff > maxWSBackoff {
		backoff = maxWSBackoff
	}
	delay := quotaRetryDelay(backoff, len(w.subs), w.outbound, time.Now(), frameBudget, time.Minute)
	w.subscribeFailRetries++
	w.armSubscribeRetryLocked(delay, true)
}

// recheck is true only for the first wake, so a replay that can never fit does not loop.
func (w *WebSocket) armSubscribeRetryLocked(delay time.Duration, recheck bool) {
	w.retryTimer = time.AfterFunc(delay, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.retryTimer = nil
		if !w.running || w.conn == nil || len(w.subs) == 0 {
			return
		}
		if recheck {
			extra := quotaRetryRemaining(len(w.subs), w.outbound, time.Now(), frameBudget, time.Minute)
			if extra > 0 {
				w.armSubscribeRetryLocked(extra, false)
				return
			}
		}
		w.resubscribeLocked(false)
	})
}

func (w *WebSocket) watchdog(ctx context.Context, conn *websocket.Conn) {
	if w.staleAfter <= 0 {
		return
	}
	tick := w.staleAfter / 3
	if tick < 20*time.Millisecond {
		tick = 20 * time.Millisecond
	}
	if tick > 15*time.Second {
		tick = 15 * time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.mu.Lock()
			if w.conn != conn || len(w.subs) == 0 {
				w.mu.Unlock()
				continue
			}
			now := time.Now()
			ackGap := watchdogResendGap(len(w.subs), w.ackTimeout)
			if w.pendingAcks > 0 && !w.lastSubscribe.IsZero() && now.Sub(w.lastSubscribe) >= ackGap {
				w.resubscribeLocked(true)
				w.mu.Unlock()
				continue
			}
			if !w.hadPush || w.lastPush.IsZero() || now.Sub(w.lastPush) < w.staleAfter {
				w.mu.Unlock()
				continue
			}
			shift := w.staleRetries
			if shift > 3 {
				shift = 3
			}
			cooldown := w.staleAfter * time.Duration(1<<shift)
			if cooldown > 5*time.Minute {
				cooldown = 5 * time.Minute
			}
			if w.staleRetries > 0 && !w.lastResub.IsZero() && now.Sub(w.lastResub) < cooldown {
				w.mu.Unlock()
				continue
			}
			w.staleRetries++
			w.lastResub = now
			w.resubscribeLocked(true)
			w.mu.Unlock()
		}
	}
}

func (w *WebSocket) resubscribeLocked(trackAcks bool) {
	w.pendingAcks = 0
	for _, sub := range w.subs {
		switch sub.kind {
		case "trade":
			w.sendLocked(codesMessage(int(WsSubTrade), sub.codes, sub.includeTy))
		case "depth":
			w.sendLocked(codesMessage(int(WsSubDepth), sub.codes, false))
		case "kline":
			w.sendLocked(klineMessage(int(WsSubKline), sub.codes, sub.klineType))
		}
		if trackAcks {
			w.noteSubscribeSentLocked()
		}
	}
	if !trackAcks {
		w.pendingAcks = 0
	}
}

func (w *WebSocket) sendLocked(payload any) error {
	if w.conn == nil {
		return nil
	}
	_ = w.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := w.conn.WriteJSON(payload)
	if err != nil {
		c := w.conn
		w.conn = nil
		go func() { _ = c.Close() }()
		return err
	}
	w.noteOutboundLocked(time.Now())
	return nil
}

func (w *WebSocket) noteOutboundLocked(at time.Time) {
	w.outbound = append(w.outbound, at)
	if over := len(w.outbound) - outboundStampLimit; over > 0 {
		w.outbound = w.outbound[over:]
	}
}

// SubscribeTrade records codes and sends 10000 when the socket is open.
func (w *WebSocket) SubscribeTrade(codes string, includeTy bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subs["trade|"+codes] = subscription{kind: "trade", codes: codes, includeTy: includeTy}
	w.sendLocked(codesMessage(int(WsSubTrade), codes, includeTy))
	w.noteSubscribeSentLocked()
}

func (w *WebSocket) SubscribeDepth(codes string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subs["depth|"+codes] = subscription{kind: "depth", codes: codes}
	w.sendLocked(codesMessage(int(WsSubDepth), codes, false))
	w.noteSubscribeSentLocked()
}

func (w *WebSocket) SubscribeKline(codes string, klineType KlineType) {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := fmt.Sprintf("kline|%s|%d", codes, klineType)
	w.subs[key] = subscription{kind: "kline", codes: codes, klineType: int(klineType)}
	w.sendLocked(klineMessage(int(WsSubKline), codes, int(klineType)))
	w.noteSubscribeSentLocked()
}

func (w *WebSocket) UnsubscribeTrade(codes string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.subs, "trade|"+codes)
	w.sendLocked(codesMessage(int(WsUnsubTrade), codes, false))
}

func (w *WebSocket) UnsubscribeDepth(codes string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.subs, "depth|"+codes)
	w.sendLocked(codesMessage(int(WsUnsubDepth), codes, false))
}

func (w *WebSocket) UnsubscribeKline(codes string, klineType KlineType) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.subs, fmt.Sprintf("kline|%s|%d", codes, klineType))
	w.sendLocked(map[string]any{
		"code":  int(WsUnsubKline),
		"trace": traceID(),
		"data":  map[string]any{"codes": codes, "klineTypes": fmt.Sprintf("%d", klineType)},
	})
}

func (w *WebSocket) stopForTerminal() {
	w.mu.Lock()
	w.running = false
	conn := w.conn
	w.conn = nil
	w.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

// Close stops reconnect and closes the socket.
func (w *WebSocket) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
	if w.stop != nil {
		select {
		case <-w.stop:
		default:
			close(w.stop)
		}
		w.stop = nil
	}
	w.stopRetryLocked()
	if w.conn != nil {
		_ = w.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client close"), time.Now().Add(time.Second))
		err := w.conn.Close()
		w.conn = nil
		return err
	}
	return nil
}

func (w *WebSocket) emitError(err error) {
	if w.OnError != nil {
		w.safeCallErr(w.OnError, err)
	}
}

func (w *WebSocket) safeCall(cb func(map[string]any), data map[string]any) {
	defer func() { _ = recover() }()
	cb(data)
}

func (w *WebSocket) safeCallStr(cb func(string), s string) {
	defer func() { _ = recover() }()
	cb(s)
}

func (w *WebSocket) safeCallErr(cb func(error), err error) {
	defer func() { _ = recover() }()
	cb(err)
}

func (w *WebSocket) safeRun(cb func()) {
	if cb == nil {
		return
	}
	defer func() { _ = recover() }()
	cb()
}

func codesMessage(code int, codes string, includeTy bool) map[string]any {
	data := map[string]any{"codes": codes}
	if includeTy {
		data["includeTy"] = true
	}
	return map[string]any{"code": code, "trace": traceID(), "data": data}
}

func klineMessage(code int, codes string, klineType int) map[string]any {
	return map[string]any{
		"code":  code,
		"trace": traceID(),
		"data":  map[string]any{"arr": []map[string]any{{"codes": codes, "type": klineType}}},
	}
}

func traceID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func wsAuthMessage(detail string) string {
	msg := "WebSocket handshake rejected with HTTP 401 — the API key is invalid or not authorised for this channel. Not reconnecting."
	if detail != "" {
		return msg + " (" + detail + ")"
	}
	return msg
}
