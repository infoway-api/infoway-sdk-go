package infoway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func startWS(t *testing.T, onConnect func(*websocket.Conn), onMessage func([]byte)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.WriteJSON(map[string]any{"code": 200, "msg": "ws connect success"})
		if onConnect != nil {
			onConnect(c)
		}
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}
			if onMessage != nil {
				onMessage(raw)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWSSubscribeTradeIncludeTy(t *testing.T) {
	got := make(chan map[string]any, 4)
	srv := startWS(t, nil, func(raw []byte) {
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		got <- m
	})
	ws, err := NewWebSocket(WSOptions{
		APIKey: "k", Business: BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	time.Sleep(80 * time.Millisecond)
	ws.SubscribeTrade("BTCUSDT", true)
	select {
	case m := <-got:
		if int(m["code"].(float64)) != 10000 {
			t.Fatalf("%v", m)
		}
		data := m["data"].(map[string]any)
		if data["codes"] != "BTCUSDT" || data["includeTy"] != true {
			t.Fatalf("%v", data)
		}
		if _, ok := m["trace"].(string); !ok {
			t.Fatal("trace")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = ws.Close()
}

func TestWSKlineAndHeartApplyIgnored(t *testing.T) {
	var conn *websocket.Conn
	ready := make(chan struct{}, 1)
	srv := startWS(t, func(c *websocket.Conn) {
		conn = c
		ready <- struct{}{}
	}, nil)
	ws, err := NewWebSocket(WSOptions{
		APIKey: "k", Business: BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan map[string]any, 4)
	ws.OnKline = func(d map[string]any) { ticks <- d }
	ws.OnError = func(e error) {
		if _, ok := e.(*APIError); ok {
			t.Errorf("heart apply must not be an API error: %v", e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	<-ready
	_ = conn.WriteJSON(map[string]any{"code": 10011, "msg": "ok"})
	_ = conn.WriteJSON(map[string]any{"code": 10008, "data": map[string]any{"s": "BTCUSDT", "ty": 1, "t": "1700000000"}})
	select {
	case d := <-ticks:
		if d["s"] != "BTCUSDT" {
			t.Fatalf("%v", d)
		}
		if _, ok := d["code"]; ok {
			t.Fatal("must unwrap data")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = ws.Close()
}

func TestWS508IsAPIKeyExpired(t *testing.T) {
	var conn *websocket.Conn
	ready := make(chan struct{}, 1)
	srv := startWS(t, func(c *websocket.Conn) {
		conn = c
		ready <- struct{}{}
	}, nil)
	ws, err := NewWebSocket(WSOptions{
		APIKey: "k", Business: BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	errs := make(chan error, 1)
	ws.OnError = func(e error) { errs <- e }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	<-ready
	_ = conn.WriteJSON(map[string]any{"code": 508, "msg": "expired", "traceId": "t"})
	select {
	case e := <-errs:
		api, ok := e.(*APIError)
		if !ok {
			t.Fatalf("%T %v", e, e)
		}
		if api.ErrorName != "APIKEY_EXPIRED" {
			t.Fatalf("%+v", api)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	_ = ws.Close()
}

func TestWS401Handshake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	t.Cleanup(srv.Close)
	ws, err := NewWebSocket(WSOptions{
		APIKey: "bad", Business: BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = ws.Connect(ctx)
	if _, ok := err.(*AuthError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestNewsSubscribeAndParsed(t *testing.T) {
	var conn *websocket.Conn
	got := make(chan map[string]any, 4)
	ready := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "business=") {
			t.Error("news must not send business")
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		conn = c
		ready <- struct{}{}
		_ = c.WriteJSON(map[string]any{"code": 200, "msg": "ws connect success"})
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			got <- m
		}
	}))
	t.Cleanup(srv.Close)
	news, err := NewNewsWebSocket(NewsOptions{
		APIKey:               "k",
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		Lang:                 "en",
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	items := make(chan map[string]any, 1)
	parsed := make(chan map[string]any, 1)
	news.OnNews = func(d map[string]any) { items <- d }
	news.OnNewsParsed = func(d map[string]any) { parsed <- d }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = news.Connect(ctx) }()
	<-ready
	select {
	case m := <-got:
		if int(m["code"].(float64)) != 10020 {
			t.Fatalf("auto subscribe %v", m)
		}
		if m["data"].(map[string]any)["lang"] != "en" {
			t.Fatalf("%v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no subscribe")
	}
	_ = conn.WriteJSON(map[string]any{
		"code": 10022,
		"data": map[string]any{"title": "hi", "published": 1786775220.0, "dk": "abc"},
	})
	select {
	case d := <-items:
		if d["published"].(float64) != 1786775220 {
			t.Fatalf("raw published %v", d["published"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no news")
	}
	select {
	case d := <-parsed:
		if _, ok := d["published"].(time.Time); !ok {
			t.Fatalf("parsed published %T", d["published"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no parsed")
	}
	news.Unsubscribe()
	select {
	case m := <-got:
		if int(m["code"].(float64)) != 11020 {
			t.Fatalf("%v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no unsub")
	}
	_ = news.Close()
}

func TestCloseWakesBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	t.Cleanup(srv.Close)
	ws, err := NewWebSocket(WSOptions{
		APIKey: "k", Business: BusinessCrypto,
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- ws.Connect(context.Background()) }()
	time.Sleep(80 * time.Millisecond)
	start := time.Now()
	_ = ws.Close()
	select {
	case <-done:
		if time.Since(start) > 2*time.Second {
			t.Fatal("Close must not wait out the reconnect backoff")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Connect did not return after Close")
	}
}

func TestNews401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	t.Cleanup(srv.Close)
	news, err := NewNewsWebSocket(NewsOptions{
		APIKey:               "bad",
		BaseURL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = news.Connect(ctx)
	if _, ok := err.(*AuthError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "news") {
		t.Fatalf("%v", err)
	}
}
