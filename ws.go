package infoway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultWSURL      = "wss://data.infoway.io/ws"
	heartbeatInterval = 30 * time.Second
	initialBackoff    = time.Second
	maxWSBackoff      = 30 * time.Second
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
	hbInterval   time.Duration
	initBackoff  time.Duration
	hbGens       int
	OnTrade      func(map[string]any)
	OnDepth      func(map[string]any)
	OnKline      func(map[string]any)
	OnFrame      func(string)
	OnError      func(error)
	OnReconnect  func()
	OnDisconnect func()

	mu        sync.Mutex
	subs      map[string]subscription
	conn      *websocket.Conn
	stop      chan struct{}
	running   bool
	connected bool
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
	return &WebSocket{
		url:          fmt.Sprintf("%s?business=%s&apikey=%s", base, opts.Business, key),
		maxReconnect: opts.MaxReconnectAttempts,
		printFrames:  opts.PrintFrames,
		hbInterval:   hb,
		initBackoff:  bo,
		subs:         map[string]subscription{},
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
	w.resubscribeLocked()
	w.mu.Unlock()
	if reconnected {
		w.safeRun(w.OnReconnect)
	}

	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	w.hbGens++
	go w.heartbeat(hbCtx, conn)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			w.mu.Lock()
			if w.conn == conn {
				w.conn = nil
			}
			w.mu.Unlock()
			cancel()
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

func (w *WebSocket) dispatch(raw []byte) {
	if w.printFrames {
		log.Print(string(raw))
	}
	if w.OnFrame != nil {
		w.safeCallStr(w.OnFrame, string(raw))
	}
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return // stock welcome is plain text
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
		return
	}
	data, _ := msg["data"].(map[string]any)
	switch WsCode(code) {
	case WsPushTrade:
		if data != nil && w.OnTrade != nil {
			w.safeCall(w.OnTrade, data)
		}
	case WsPushDepth:
		if data != nil && w.OnDepth != nil {
			w.safeCall(w.OnDepth, data)
		}
	case WsPushKline:
		if data != nil && w.OnKline != nil {
			w.safeCall(w.OnKline, data)
		}
	}
}

func (w *WebSocket) resubscribeLocked() {
	for _, sub := range w.subs {
		switch sub.kind {
		case "trade":
			w.sendLocked(codesMessage(int(WsSubTrade), sub.codes, sub.includeTy))
		case "depth":
			w.sendLocked(codesMessage(int(WsSubDepth), sub.codes, false))
		case "kline":
			w.sendLocked(klineMessage(int(WsSubKline), sub.codes, sub.klineType))
		}
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
	}
	return err
}

// SubscribeTrade records codes and sends 10000 when the socket is open.
func (w *WebSocket) SubscribeTrade(codes string, includeTy bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subs["trade|"+codes] = subscription{kind: "trade", codes: codes, includeTy: includeTy}
	w.sendLocked(codesMessage(int(WsSubTrade), codes, includeTy))
}

func (w *WebSocket) SubscribeDepth(codes string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subs["depth|"+codes] = subscription{kind: "depth", codes: codes}
	w.sendLocked(codesMessage(int(WsSubDepth), codes, false))
}

func (w *WebSocket) SubscribeKline(codes string, klineType KlineType) {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := fmt.Sprintf("kline|%s|%d", codes, klineType)
	w.subs[key] = subscription{kind: "kline", codes: codes, klineType: int(klineType)}
	w.sendLocked(klineMessage(int(WsSubKline), codes, int(klineType)))
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
	if w.conn != nil {
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
