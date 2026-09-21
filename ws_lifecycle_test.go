package infoway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type lifeServer struct {
	connections atomic.Int32
	mu          sync.Mutex
	received    []string
	clients     []*websocket.Conn
}

func (l *lifeServer) codes() []int {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]int, 0, len(l.received))
	for _, raw := range l.received {
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) != nil {
			continue
		}
		if n, ok := asInt(m["code"]); ok {
			out = append(out, n)
		}
	}
	return out
}

func (l *lifeServer) count(code int) int {
	n := 0
	for _, c := range l.codes() {
		if c == code {
			n++
		}
	}
	return n
}

func (l *lifeServer) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.received...)
}

func (l *lifeServer) resetReceived() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.received = nil
}

func (l *lifeServer) dropAll() {
	l.mu.Lock()
	cs := append([]*websocket.Conn(nil), l.clients...)
	l.clients = nil
	l.mu.Unlock()
	for _, c := range cs {
		_ = c.Close()
	}
}

func startLife(t *testing.T) (*httptest.Server, *lifeServer) {
	t.Helper()
	l := &lifeServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		l.connections.Add(1)
		l.mu.Lock()
		l.clients = append(l.clients, c)
		l.mu.Unlock()
		_ = c.WriteJSON(map[string]any{"code": 200, "msg": "ws connect success"})
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}
			l.mu.Lock()
			l.received = append(l.received, string(raw))
			l.mu.Unlock()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, l
}

func waitUntil(t *testing.T, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met")
}

func lifeClient(t *testing.T, url string) *WebSocket {
	t.Helper()
	ws, err := NewWebSocket(WSOptions{
		APIKey:            "k",
		Business:          BusinessCrypto,
		BaseURL:           url,
		HeartbeatInterval: 80 * time.Millisecond,
		ReconnectBackoff:  20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestLifecycleDropReconnectsOnceAndResubscribes(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	var reconnects, disconnects atomic.Int32
	var ticks []map[string]any
	var tickMu sync.Mutex
	ws.OnReconnect = func() { reconnects.Add(1) }
	ws.OnDisconnect = func() { disconnects.Add(1) }
	ws.OnTrade = func(d map[string]any) {
		tickMu.Lock()
		ticks = append(ticks, d)
		tickMu.Unlock()
	}
	ws.SubscribeTrade("BTCUSDT", false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()

	waitUntil(t, func() bool { return life.connections.Load() == 1 && life.count(int(WsSubTrade)) == 1 })
	if reconnects.Load() != 0 {
		t.Fatalf("first connect fired OnReconnect")
	}

	life.dropAll()
	waitUntil(t, func() bool { return life.connections.Load() == 2 && life.count(int(WsSubTrade)) >= 2 })
	if reconnects.Load() != 1 {
		t.Fatalf("reconnects=%d", reconnects.Load())
	}
	if disconnects.Load() != 1 {
		t.Fatalf("disconnects=%d", disconnects.Load())
	}
	if ws.hbGens != 2 {
		t.Fatalf("hbGens=%d", ws.hbGens)
	}

	life.mu.Lock()
	if len(life.clients) == 0 {
		life.mu.Unlock()
		t.Fatal("no live client")
	}
	_ = life.clients[0].WriteJSON(map[string]any{"code": 10002, "data": map[string]any{"s": "BTCUSDT", "p": "1"}})
	life.mu.Unlock()
	waitUntil(t, func() bool {
		tickMu.Lock()
		defer tickMu.Unlock()
		return len(ticks) == 1
	})
	time.Sleep(80 * time.Millisecond)
	if life.connections.Load() != 2 {
		t.Fatalf("extra reconnect: %d", life.connections.Load())
	}
	_ = ws.Close()
}

func TestLifecycleUserCloseDoesNotReconnect(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	var reconnects, disconnects atomic.Int32
	ws.OnReconnect = func() { reconnects.Add(1) }
	ws.OnDisconnect = func() { disconnects.Add(1) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ws.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 })
	if err := ws.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Connect did not return after Close")
	}
	time.Sleep(150 * time.Millisecond)
	if life.connections.Load() != 1 {
		t.Fatalf("reconnected after close: %d", life.connections.Load())
	}
	if disconnects.Load() != 0 || reconnects.Load() != 0 {
		t.Fatalf("disconnects=%d reconnects=%d", disconnects.Load(), reconnects.Load())
	}
}

func TestLifecycleUnsubscribeNotReplayed(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 })
	ws.SubscribeTrade("BTCUSDT", false)
	ws.SubscribeDepth("BTCUSDT")
	waitUntil(t, func() bool { return life.count(int(WsSubDepth)) >= 1 })
	ws.UnsubscribeTrade("BTCUSDT")
	waitUntil(t, func() bool { return life.count(int(WsUnsubTrade)) >= 1 })
	life.resetReceived()
	life.dropAll()
	waitUntil(t, func() bool { return life.connections.Load() == 2 && len(life.snapshot()) >= 1 })
	time.Sleep(40 * time.Millisecond)
	if life.count(int(WsSubDepth)) < 1 {
		t.Fatal("depth not replayed")
	}
	if life.count(int(WsSubTrade)) != 0 {
		t.Fatal("trade was replayed after unsubscribe")
	}
	_ = ws.Close()
}

func TestLifecycleKlineIntervalKept(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 })
	ws.SubscribeKline("BTCUSDT", KlineMin1)
	ws.SubscribeKline("BTCUSDT", KlineDay)
	ws.UnsubscribeKline("BTCUSDT", KlineMin1)
	waitUntil(t, func() bool { return life.count(int(WsUnsubKline)) >= 1 })
	life.resetReceived()
	life.dropAll()
	waitUntil(t, func() bool { return life.connections.Load() == 2 && len(life.snapshot()) >= 1 })
	time.Sleep(40 * time.Millisecond)
	kline := 0
	for _, raw := range life.snapshot() {
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) != nil {
			continue
		}
		if n, ok := asInt(m["code"]); !ok || n != int(WsSubKline) {
			continue
		}
		kline++
		data := m["data"].(map[string]any)
		arr := data["arr"].([]any)
		item := arr[0].(map[string]any)
		if as, _ := asInt(item["type"]); as != int(KlineDay) {
			t.Fatalf("replayed type=%v", item["type"])
		}
	}
	if kline != 1 {
		t.Fatalf("kline subs=%d", kline)
	}
	_ = ws.Close()
}

func TestLifecycleSubscribeWhileOfflineFlushed(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 })
	life.dropAll()
	ws.SubscribeTrade("ETHUSDT", false)
	waitUntil(t, func() bool { return life.connections.Load() == 2 && life.count(int(WsSubTrade)) >= 1 })
	found := false
	for _, raw := range life.snapshot() {
		if strings.Contains(raw, "ETHUSDT") && strings.Contains(raw, "10000") {
			found = true
		}
	}
	if !found {
		t.Fatalf("offline subscribe not flushed: %v", life.snapshot())
	}
	_ = ws.Close()
}

func TestLifecycleCloseDuringBackoff(t *testing.T) {
	srv, life := startLife(t)
	ws := lifeClient(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ws.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 })
	life.dropAll()
	time.Sleep(5 * time.Millisecond)
	start := time.Now()
	if err := ws.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not wake backoff")
	}
	if time.Since(start) > 400*time.Millisecond {
		t.Fatal("Close blocked on backoff")
	}
	time.Sleep(80 * time.Millisecond)
	if life.connections.Load() != 1 {
		t.Fatalf("reconnected after close: %d", life.connections.Load())
	}
}

func TestLifecycleHandshakeHonorsMaxReconnect(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "no", 500)
	}))
	t.Cleanup(srv.Close)
	ws, err := NewWebSocket(WSOptions{
		APIKey:               "k",
		Business:             BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
		ReconnectBackoff:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = ws.Connect(ctx)
	if hits.Load() != 2 {
		t.Fatalf("hits=%d want 2", hits.Load())
	}
}

func TestLifecycleNewsDropReplaysLang(t *testing.T) {
	srv, life := startLife(t)
	n, err := NewNewsWebSocket(NewsOptions{
		APIKey:            "k",
		BaseURL:           "ws" + strings.TrimPrefix(srv.URL, "http"),
		Lang:              "en",
		HeartbeatInterval: 80 * time.Millisecond,
		ReconnectBackoff:  20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	var reconnects atomic.Int32
	n.OnReconnect = func() { reconnects.Add(1) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = n.Connect(ctx) }()
	waitUntil(t, func() bool { return life.connections.Load() == 1 && strings.Contains(strings.Join(life.snapshot(), "\n"), `"lang":"en"`) })
	if reconnects.Load() != 0 {
		t.Fatal("news first connect fired OnReconnect")
	}
	life.resetReceived()
	life.dropAll()
	waitUntil(t, func() bool {
		return life.connections.Load() == 2 && strings.Contains(strings.Join(life.snapshot(), "\n"), `"lang":"en"`)
	})
	if reconnects.Load() != 1 {
		t.Fatalf("news reconnects=%d", reconnects.Load())
	}
	_ = n.Close()
}
