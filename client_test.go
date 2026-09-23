package infoway

import "testing"

func TestVersion(t *testing.T) {
	if Version != "0.4.1" {
		t.Fatalf("Version=%s", Version)
	}
}

func TestClientSubclients(t *testing.T) {
	c := New(Options{APIKey: "k"})
	if c.Stock == nil || c.Crypto == nil || c.Japan == nil || c.India == nil ||
		c.Korea == nil || c.Taiwan == nil || c.Common == nil || c.Basic == nil ||
		c.Packages == nil || c.Market == nil || c.Plate == nil || c.StockInfo == nil ||
		c.Financial == nil {
		t.Fatal("missing subclient")
	}
	if c.Stock.prefix != "stock" || c.Korea.prefix != "korea" || c.Taiwan.prefix != "taiwan" {
		t.Fatalf("prefixes %s %s %s", c.Stock.prefix, c.Korea.prefix, c.Taiwan.prefix)
	}
}

func TestAPIKeyFromEnv(t *testing.T) {
	t.Setenv("INFOWAY_API_KEY", "env-key")
	c := New(Options{})
	if c.http.apiKey != "env-key" {
		t.Fatalf("apiKey=%q", c.http.apiKey)
	}
}

func TestRequireAPIKey(t *testing.T) {
	t.Setenv("INFOWAY_API_KEY", "")
	if _, err := NewWebSocket(WSOptions{Business: BusinessCrypto}); err == nil {
		t.Fatal("expected error")
	}
}

func TestJoinMarkets(t *testing.T) {
	if got := JoinMarkets(MarketHK, MarketUS); got != "HK,US" {
		t.Fatal(got)
	}
}

func TestParseScheduleType(t *testing.T) {
	if _, ok := ParseScheduleType("ENERGY"); !ok {
		t.Fatal("ENERGY")
	}
	if _, ok := ParseScheduleType("STOCK_US"); ok {
		t.Fatal("STOCK_US must fail")
	}
}

func TestErrorNamesCollide(t *testing.T) {
	if RestErrorName(508) != "PRODUCT_NOT_EXISTS" {
		t.Fatal(RestErrorName(508))
	}
	if WsErrorName(508) != "APIKEY_EXPIRED" {
		t.Fatal(WsErrorName(508))
	}
	if !IsWsError(508) || IsWsError(10002) || IsWsError(10011) {
		t.Fatal("IsWsError")
	}
}
