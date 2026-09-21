package infoway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHTTP(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(Options{APIKey: "test-key", BaseURL: srv.URL, MaxRetries: 1, Timeout: defaultTimeout})
	t.Cleanup(c.Close)
	return c, srv
}

func TestHTTPStandardEnvelope(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apiKey") != "test-key" {
			t.Errorf("apiKey header = %q", r.Header.Get("apiKey"))
		}
		_, _ = io.WriteString(w, `{"ret":200,"msg":"ok","traceId":"t1","data":[{"s":"AAPL.US","p":"305.771","t":1786751999691,"vw":"305.771","v":"1","td":1}]}`)
	})
	data, err := c.Stock.GetTrade(context.Background(), "AAPL.US")
	if err != nil {
		t.Fatal(err)
	}
	rows := data.([]any)
	row := rows[0].(map[string]any)
	if row["s"] != "AAPL.US" {
		t.Fatalf("s=%v", row["s"])
	}
	if row["p"] != "305.771" {
		t.Fatalf("p=%v", row["p"])
	}
}

func TestHTTPV2Envelope(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"market":"HK","count":1,"data":[{"symbol":"IN20234.HK"}]}`)
	})
	data, err := c.Plate.GetIndustry(context.Background(), "HK", 10)
	if err != nil {
		t.Fatal(err)
	}
	rows := data.([]any)
	if rows[0].(map[string]any)["symbol"] != "IN20234.HK" {
		t.Fatalf("%v", data)
	}
}

func TestHTTPNoDataKey(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"plate":"IN20293.HK","intro":"A long enough intro text"}`)
	})
	data, err := c.Plate.GetIntro(context.Background(), "IN20293.HK")
	if err != nil {
		t.Fatal(err)
	}
	obj := data.(map[string]any)
	if obj["plate"] != "IN20293.HK" {
		t.Fatalf("%v", data)
	}
}

func TestHTTPRFC7807(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Required parameter 'type' is not present."}`)
	})
	_, err := c.Basic.GetSymbols(context.Background(), "", "")
	api, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if api.Ret != 400 || api.Msg != "Required parameter 'type' is not present." {
		t.Fatalf("%+v", api)
	}
	if api.ErrorName != "" {
		t.Fatalf("http-status must not set ErrorName, got %s", api.ErrorName)
	}
}

func TestHTTPRateLimitDetail200(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"detail":"Rate limit exceeded"}`)
	})
	_, err := c.StockInfo.GetRatings(context.Background(), "AAPL.US", "")
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestHTTP401(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = io.WriteString(w, `{"message":"Token invalid"}`)
	})
	_, err := c.Crypto.GetTrade(context.Background(), "BTCUSDT")
	auth, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if auth.Msg != "Token invalid" {
		t.Fatalf("%v", auth)
	}
}

func TestHTTPRest508ProductNotExists(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"ret":508,"msg":"All product not exists","traceId":"x"}`)
	})
	_, err := c.Stock.GetTrade(context.Background(), "NOPE.US")
	api, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if api.Ret != 508 || api.ErrorName != "PRODUCT_NOT_EXISTS" {
		t.Fatalf("%+v", api)
	}
}

func TestHTTPRest501RateLimit(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ret":501,"msg":"Request frequency exceed the limit","traceId":"x"}`)
	})
	_, err := c.Crypto.GetTrade(context.Background(), "BTCUSDT")
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestHTTPBusinessRetBeforeHTTP400(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"ret":400,"msg":"not support type","traceId":"x"}`)
	})
	_, err := c.Basic.GetSymbols(context.Background(), "US", "")
	api, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if api.ErrorName != "BAD_REQUEST" || api.Msg != "not support type" {
		t.Fatalf("%+v", api)
	}
}

func TestHTTPKlineBody(t *testing.T) {
	var got map[string]any
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.WriteString(w, `{"ret":200,"data":[{"s":"BTCUSDT","respList":[]}]}`)
	})
	ts := int64(1_700_000_000)
	_, err := c.Crypto.GetKline(context.Background(), "BTCUSDT", KlineMin1, 2, &ts)
	if err != nil {
		t.Fatal(err)
	}
	if got["codes"] != "BTCUSDT" || got["klineType"].(float64) != 1 || got["klineNum"].(float64) != 2 {
		t.Fatalf("%v", got)
	}
	if got["timestamp"].(float64) != 1_700_000_000 {
		t.Fatalf("timestamp %v", got["timestamp"])
	}
}

func TestHTTPEmptyBody400(t *testing.T) {
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	})
	_, err := c.Crypto.GetTrade(context.Background(), "BTCUSDT")
	api, ok := err.(*APIError)
	if !ok {
		t.Fatalf("got %T %v", err, err)
	}
	if api.Ret != 502 || api.ErrorName != "" {
		t.Fatalf("HTTP 502 must not map RestErrorCode, got %+v", api)
	}
}
