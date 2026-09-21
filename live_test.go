package infoway

import (
	"context"
	"os"
	"testing"
	"time"
)

func liveEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("INFOWAY_LIVE") != "1" || os.Getenv("INFOWAY_API_KEY") == "" {
		t.Skip("set INFOWAY_LIVE=1 and INFOWAY_API_KEY")
	}
}

func liveClient(t *testing.T) *Client {
	t.Helper()
	liveEnabled(t)
	c := New(Options{APIKey: os.Getenv("INFOWAY_API_KEY"), MaxRetries: 2})
	t.Cleanup(c.Close)
	return c
}

func TestLiveCryptoTrade(t *testing.T) {
	c := liveClient(t)
	data, err := c.Crypto.GetTrade(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	row := data.([]any)[0].(map[string]any)
	if row["s"] != "BTCUSDT" {
		t.Fatalf("%v", row)
	}
}

func TestLiveCryptoDepthAndKline(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	depth, err := c.Crypto.GetDepth(ctx, "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	book := depth.([]any)[0].(map[string]any)
	if _, ok := book["asks"]; ok {
		t.Fatal("asks")
	}
	a := book["a"].([]any)
	if len(a) != 2 {
		t.Fatalf("a=%v", a)
	}
	klines, err := c.Crypto.GetKline(ctx, "BTCUSDT", KlineMin1, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	row := klines.([]any)[0].(map[string]any)
	if _, ok := row["kline"]; ok {
		t.Fatal("kline")
	}
	if _, ok := row["respList"]; !ok {
		t.Fatalf("%v", row)
	}
}

func TestLiveKoreaTaiwanPackagesFinancial(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	if _, err := c.Korea.GetTrade(ctx, "005930.KS"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Taiwan.GetTrade(ctx, "2330.TW"); err != nil {
		t.Fatal(err)
	}
	pkg, err := c.Packages.GetInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	m := pkg.(map[string]any)
	if m["packageName"] == nil && m["package_name"] == nil {
		t.Fatalf("%v", pkg)
	}
	if _, err := c.Financial.GetEarningStatus(ctx, "AAPL.US", SymbolStockUS); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Market.GetRankCategories(ctx, "US", ""); err != nil {
		t.Fatal(err)
	}
}

func TestLiveUnknownSymbol(t *testing.T) {
	c := liveClient(t)
	_, err := c.Stock.GetTrade(context.Background(), "NOSUCH.US")
	if err == nil {
		t.Fatal("expected error")
	}
	if api, ok := err.(*APIError); !ok || api.Ret == 0 {
		t.Fatalf("%T %v", err, err)
	}
}

func TestLiveForgedKey(t *testing.T) {
	liveEnabled(t)
	bad := New(Options{APIKey: "invalid-key-sdk-deep-test-000000-infoway", MaxRetries: 1})
	defer bad.Close()
	_, err := bad.Crypto.GetTrade(context.Background(), "BTCUSDT")
	if _, ok := err.(*AuthError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestLiveWSCryptoTrade(t *testing.T) {
	liveEnabled(t)
	ws, err := NewWebSocket(WSOptions{
		APIKey:               os.Getenv("INFOWAY_API_KEY"),
		Business:             BusinessCrypto,
		MaxReconnectAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan map[string]any, 1)
	ws.OnTrade = func(d map[string]any) {
		select {
		case ticks <- d:
		default:
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	ws.SubscribeTrade("BTCUSDT", false)
	select {
	case d := <-ticks:
		if d["s"] != "BTCUSDT" {
			t.Fatalf("%v", d)
		}
		if _, ok := d["code"]; ok {
			t.Fatal("envelope")
		}
	case <-ctx.Done():
		t.Fatal("no tick")
	}
	_ = ws.Close()
}
