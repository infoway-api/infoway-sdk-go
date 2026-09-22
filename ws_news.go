package infoway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const defaultNewsURL = "wss://data.infoway.io/news"

// NewsOptions configures the news channel.
type NewsOptions struct {
	APIKey               string
	BaseURL              string
	Lang                 string // default en
	MaxReconnectAttempts int
	PrintFrames          bool
	HeartbeatInterval    time.Duration
	ReconnectBackoff     time.Duration
}

// NewsWebSocket is wss://data.infoway.io/news?apikey=… (separate entitlement).
type NewsWebSocket struct {
	url          string
	maxReconnect int
	printFrames  bool
	hbInterval   time.Duration
	initBackoff  time.Duration
	OnNews       func(map[string]any)
	OnNewsParsed func(map[string]any)
	OnFrame      func(string)
	OnError      func(error)
	OnReconnect  func()
	OnDisconnect func()

	mu        sync.Mutex
	lang      string
	conn      *websocket.Conn
	stop      chan struct{}
	running   bool
	connected bool
}

func NewNewsWebSocket(opts NewsOptions) (*NewsWebSocket, error) {
	key, err := requireAPIKey(opts.APIKey)
	if err != nil {
		return nil, err
	}
	base := opts.BaseURL
	if base == "" {
		base = defaultNewsURL
	}
	lang := opts.Lang
	if lang == "" {
		lang = "en"
	}
	hb := opts.HeartbeatInterval
	if hb <= 0 {
		hb = heartbeatInterval
	}
	bo := opts.ReconnectBackoff
	if bo <= 0 {
		bo = initialBackoff
	}
	return &NewsWebSocket{
		url:          fmt.Sprintf("%s?apikey=%s", base, key),
		maxReconnect: opts.MaxReconnectAttempts,
		printFrames:  opts.PrintFrames,
		hbInterval:   hb,
		initBackoff:  bo,
		lang:         lang,
	}, nil
}

func (n *NewsWebSocket) Connect(ctx context.Context) error {
	n.mu.Lock()
	n.running = true
	stop := make(chan struct{})
	n.stop = stop
	n.mu.Unlock()

	backoff := n.initBackoff
	if backoff <= 0 {
		backoff = initialBackoff
	}
	attempt := 0
	for {
		n.mu.Lock()
		running := n.running
		n.mu.Unlock()
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

		opened, err := n.connectOnce(ctx)
		if opened {
			backoff = n.initBackoff
			attempt = 0
		}
		n.mu.Lock()
		running = n.running
		n.mu.Unlock()
		if !running {
			return nil
		}
		if err != nil {
			if _, ok := err.(*AuthError); ok {
				n.emitError(err)
				return err
			}
			n.emitError(err)
		}
		attempt++
		if n.maxReconnect > 0 && attempt > n.maxReconnect {
			return err
		}
		if n.OnDisconnect != nil {
			func() { defer func() { _ = recover() }(); n.OnDisconnect() }()
		}
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

func (n *NewsWebSocket) connectOnce(ctx context.Context) (opened bool, err error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		ReadBufferSize:   16 * 1024,
		WriteBufferSize:  4 * 1024,
	}
	conn, resp, err := dialer.DialContext(ctx, n.url, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return false, newAuthError(newsAuthMessage(), "")
		}
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			return false, OfHTTPStatus(resp.StatusCode, fmt.Sprintf("WebSocket handshake failed with HTTP %d", resp.StatusCode), "")
		}
		return false, err
	}

	n.mu.Lock()
	reconnected := n.connected
	n.conn = conn
	n.connected = true
	n.sendSubscribeLocked()
	n.mu.Unlock()
	if reconnected && n.OnReconnect != nil {
		func() { defer func() { _ = recover() }(); n.OnReconnect() }()
	}

	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go n.heartbeat(hbCtx, conn)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			n.mu.Lock()
			if n.conn == conn {
				n.conn = nil
			}
			n.mu.Unlock()
			cancel()
			return true, err
		}
		n.dispatch(raw)
	}
}

func (n *NewsWebSocket) heartbeat(ctx context.Context, conn *websocket.Conn) {
	iv := n.hbInterval
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
			n.mu.Lock()
			if n.conn != conn {
				n.mu.Unlock()
				return
			}
			_ = n.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			err := n.conn.WriteJSON(map[string]any{"code": int(WsHeartbeat), "trace": traceID()})
			n.mu.Unlock()
			if err != nil {
				_ = conn.Close()
				return
			}
		}
	}
}

func (n *NewsWebSocket) dispatch(raw []byte) {
	if n.printFrames {
		log.Print(string(raw))
	}
	if n.OnFrame != nil {
		func() { defer func() { _ = recover() }(); n.OnFrame(string(raw)) }()
	}
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	code, ok := asInt(msg["code"])
	if !ok {
		return
	}
	if code == int(WsConnectOK) || code == int(WsHeartApply) || code == int(WsSubNewsAck) {
		return
	}
	if IsWsError(code) {
		trace := strField(msg, "traceId")
		if trace == "" {
			trace = strField(msg, "trace")
		}
		n.emitError(wsFailure(code, strField(msg, "msg"), trace))
		if IsTerminalWs(code) {
			n.stopForTerminal()
		}
		return
	}
	if WsCode(code) == WsPushNews {
		data, _ := msg["data"].(map[string]any)
		if data == nil {
			return
		}
		if n.OnNews != nil {
			func() { defer func() { _ = recover() }(); n.OnNews(data) }()
		}
		if n.OnNewsParsed != nil {
			parsed := make(map[string]any, len(data))
			for k, v := range data {
				parsed[k] = v
			}
			if sec, ok := asInt(parsed["published"]); ok {
				parsed["published"] = time.Unix(int64(sec), 0).UTC()
			}
			func() { defer func() { _ = recover() }(); n.OnNewsParsed(parsed) }()
		}
	}
}

func (n *NewsWebSocket) sendSubscribeLocked() {
	if n.conn == nil || n.lang == "" {
		return
	}
	_ = n.conn.WriteJSON(map[string]any{
		"code":  int(WsSubNews),
		"trace": traceID(),
		"data":  map[string]any{"lang": n.lang},
	})
}

func (n *NewsWebSocket) Subscribe(lang string) {
	if lang == "" {
		lang = "en"
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lang = lang
	n.sendSubscribeLocked()
}

func (n *NewsWebSocket) Unsubscribe() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lang = ""
	if n.conn != nil {
		_ = n.conn.WriteJSON(map[string]any{"code": int(WsUnsubNews), "trace": traceID()})
	}
}

func (n *NewsWebSocket) stopForTerminal() {
	n.mu.Lock()
	n.running = false
	conn := n.conn
	n.conn = nil
	n.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

func (n *NewsWebSocket) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.running = false
	if n.stop != nil {
		select {
		case <-n.stop:
		default:
			close(n.stop)
		}
		n.stop = nil
	}
	if n.conn != nil {
		err := n.conn.Close()
		n.conn = nil
		return err
	}
	return nil
}

func (n *NewsWebSocket) emitError(err error) {
	if n.OnError != nil {
		func() { defer func() { _ = recover() }(); n.OnError(err) }()
	}
}

func newsAuthMessage() string {
	return "News channel handshake rejected with HTTP 401: this API key has no news permission (or is invalid). The /news route is separately authorised — contact Infoway to enable it. Not reconnecting."
}
