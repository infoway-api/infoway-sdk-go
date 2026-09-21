package infoway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type probeRow struct {
	name, expect, detail string
	ok                   bool
}

type probe struct {
	good, bad string
	rows      []probeRow
}

func (p *probe) expectOK(name string, fn func() (string, error)) {
	detail, err := fn()
	if err != nil && isTransient(err) {
		time.Sleep(400 * time.Millisecond)
		detail, err = fn()
	}
	if err != nil {
		p.rows = append(p.rows, probeRow{name, "OK", err.Error(), false})
		return
	}
	p.rows = append(p.rows, probeRow{name, "OK", detail, true})
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*TimeoutError); ok {
		return true
	}
	if _, ok := err.(*IOError); ok {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "timeout") || strings.Contains(s, "deadline")
}

func (p *probe) expectAPI(name string, fn func() error) {
	err := fn()
	if err == nil {
		p.rows = append(p.rows, probeRow{name, "API_ERROR", "succeeded unexpectedly", false})
		return
	}
	if _, ok := err.(*AuthError); ok {
		p.rows = append(p.rows, probeRow{name, "API_ERROR", "got AUTH: " + err.Error(), false})
		return
	}
	if _, ok := err.(*APIError); ok {
		p.rows = append(p.rows, probeRow{name, "API_ERROR", err.Error(), true})
		return
	}
	p.rows = append(p.rows, probeRow{name, "API_ERROR", err.Error(), false})
}

func (p *probe) expectAuth(name string, fn func() error) {
	err := fn()
	if err == nil {
		p.rows = append(p.rows, probeRow{name, "AUTH", "succeeded unexpectedly", false})
		return
	}
	ok := false
	if _, is := err.(*AuthError); is {
		ok = true
	} else if api, is := err.(*APIError); is {
		msg := strings.ToLower(api.Msg)
		ok = api.Ret == 401 || strings.Contains(msg, "apikey") || strings.Contains(msg, "token") ||
			strings.Contains(msg, "unauthorized") || strings.Contains(msg, "not exists")
	}
	p.rows = append(p.rows, probeRow{name, "AUTH", err.Error(), ok})
}

func nonempty(data any) (string, error) {
	if data == nil {
		return "", errf("null data")
	}
	if arr, ok := data.([]any); ok && len(arr) == 0 {
		return "", errf("empty array")
	}
	s := stringify(data)
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return s, nil
}

func present(data any) (string, error) {
	if data == nil {
		return "", errf("null data")
	}
	s := stringify(data)
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return s, nil
}

func errf(msg string) error { return fmt.Errorf("%s", msg) }

func stringify(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestLiveDeepProbe(t *testing.T) {
	liveEnabled(t)
	good := os.Getenv("INFOWAY_API_KEY")
	bad := os.Getenv("INFOWAY_BAD_API_KEY")
	if bad == "" {
		bad = "invalid-key-sdk-deep-test-000000-infoway"
	}
	p := &probe{good: good, bad: bad}
	ctx := context.Background()
	c := New(Options{APIKey: good, MaxRetries: 2, Timeout: 30 * time.Second})
	defer c.Close()

	p.expectOK("REST crypto.trade BTCUSDT", func() (string, error) {
		d, err := c.Crypto.GetTrade(ctx, "BTCUSDT")
		if err != nil {
			return "", err
		}
		return nonempty(d)
	})
	p.expectOK("REST crypto.depth BTCUSDT", func() (string, error) {
		d, err := c.Crypto.GetDepth(ctx, "BTCUSDT")
		if err != nil {
			return "", err
		}
		return nonempty(d)
	})
	p.expectOK("REST crypto.kline MIN_1 x2", func() (string, error) {
		d, err := c.Crypto.GetKline(ctx, "BTCUSDT", KlineMin1, 2, nil)
		if err != nil {
			return "", err
		}
		return nonempty(d)
	})
	p.expectOK("REST crypto.kline + timestamp", func() (string, error) {
		ts := int64(1_700_000_000)
		d, err := c.Crypto.GetKline(ctx, "BTCUSDT", KlineMin1, 2, &ts)
		if err != nil {
			return "", err
		}
		return nonempty(d)
	})
	for _, item := range []struct {
		name string
		fn   func() (any, error)
	}{
		{"REST stock.trade AAPL.US", func() (any, error) { return c.Stock.GetTrade(ctx, "AAPL.US") }},
		{"REST stock.trade 00700.HK", func() (any, error) { return c.Stock.GetTrade(ctx, "00700.HK") }},
		{"REST stock.trade 600519.SH", func() (any, error) { return c.Stock.GetTrade(ctx, "600519.SH") }},
		{"REST japan.trade 7203.JP", func() (any, error) { return c.Japan.GetTrade(ctx, "7203.JP") }},
		{"REST india.trade RELIANCE.IN", func() (any, error) { return c.India.GetTrade(ctx, "RELIANCE.IN") }},
		{"REST korea.trade 005930.KS", func() (any, error) { return c.Korea.GetTrade(ctx, "005930.KS") }},
		{"REST taiwan.trade 2330.TW", func() (any, error) { return c.Taiwan.GetTrade(ctx, "2330.TW") }},
		{"REST common.trade USDJPY", func() (any, error) { return c.Common.GetTrade(ctx, "USDJPY") }},
		{"REST basic.symbols CRYPTO", func() (any, error) { return c.Basic.GetSymbols(ctx, SymbolCrypto, "") }},
		{"REST basic.symbols STOCK_TW", func() (any, error) { return c.Basic.GetSymbols(ctx, SymbolStockTW, "2330.TW") }},
		{"REST basic.symbolInfo AAPL.US", func() (any, error) { return c.Basic.GetSymbolInfo(ctx, SymbolStockUS, "AAPL.US") }},
		{"REST basic.stockDetail AAPL.US", func() (any, error) { return c.Basic.GetStockDetail(ctx, SymbolStockUS, "AAPL.US") }},
		{"REST basic.markets", func() (any, error) { return c.Basic.GetMarkets(ctx) }},
		{"REST basic.tradingDays US", func() (any, error) { return c.Basic.GetTradingDays(ctx, "US", "20260801", "20260815") }},
		{"REST basic.tradingSchedule", func() (any, error) { return c.Basic.GetTradingSchedule(ctx) }},
		{"REST basic.tradingSchedule ENERGY", func() (any, error) { return c.Basic.GetTradingScheduleByType(ctx, "ENERGY") }},
		{"REST packages.info", func() (any, error) { return c.Packages.GetInfo(ctx) }},
		{"REST market.temperature HK,US", func() (any, error) { return c.Market.GetTemperature(ctx, "HK,US", "") }},
		{"REST market.breadth US", func() (any, error) { return c.Market.GetBreadth(ctx, "US", "") }},
		{"REST market.turnover US", func() (any, error) { return c.Market.GetTurnover(ctx, "US", "") }},
		{"REST market.indexes", func() (any, error) { return c.Market.GetIndexes(ctx, "") }},
		{"REST market.overview US", func() (any, error) { return c.Market.GetOverview(ctx, "US", "en") }},
		{"REST market.rank US", func() (any, error) {
			return c.Market.GetRank(ctx, "US", "all", &RankOptions{Sort: "chg", Order: "desc", Limit: 5, Lang: "en"})
		}},
		{"REST plate.industry HK", func() (any, error) { return c.Plate.GetIndustry(ctx, "HK", 10) }},
		{"REST stockInfo.company zh-CN", func() (any, error) { return c.StockInfo.GetCompany(ctx, "AAPL.US", "zh-CN") }},
		{"REST stockInfo.valuation", func() (any, error) { return c.StockInfo.GetValuation(ctx, "AAPL.US", "") }},
		{"REST financial.earningStatus", func() (any, error) { return c.Financial.GetEarningStatus(ctx, "AAPL.US", SymbolStockUS) }},
		{"REST financial.income fq", func() (any, error) {
			return c.Financial.GetIncomeStatement(ctx, "AAPL.US", SymbolStockUS, PeriodFQ)
		}},
		{"REST financial.dividend", func() (any, error) { return c.Financial.GetDividend(ctx, "AAPL.US", SymbolStockUS, "") }},
	} {
		item := item
		p.expectOK(item.name, func() (string, error) {
			d, err := item.fn()
			if err != nil {
				return "", err
			}
			if strings.Contains(item.name, "stockDetail") || strings.Contains(item.name, "markets") ||
				strings.Contains(item.name, "packages") || strings.Contains(item.name, "temperature") ||
				strings.Contains(item.name, "breadth") || strings.Contains(item.name, "turnover") ||
				strings.Contains(item.name, "indexes") || strings.Contains(item.name, "overview") ||
				strings.Contains(item.name, "rank") || strings.Contains(item.name, "plate") ||
				strings.Contains(item.name, "stockInfo") || strings.Contains(item.name, "financial") ||
				strings.Contains(item.name, "trading") {
				return present(d)
			}
			return nonempty(d)
		})
	}

	_, err := c.Basic.GetTradingScheduleByType(ctx, "STOCK_US")
	p.rows = append(p.rows, probeRow{"CLIENT scheduleByType STOCK_US", "Error", errString(err), err != nil && strings.Contains(errString(err), "ENERGY/FOREX")})

	p.expectAPI("REST trade unknown symbol", func() error {
		_, e := c.Crypto.GetTrade(ctx, "NOPE_SYMBOL_XYZ_NOT_LISTED")
		return e
	})
	p.expectAPI("REST stock trade NOPE.US", func() error { _, e := c.Stock.GetTrade(ctx, "NOPE.US"); return e })
	p.expectAPI("REST korea trade AAPL.US (wrong market)", func() error { _, e := c.Korea.GetTrade(ctx, "AAPL.US"); return e })
	p.expectAPI("REST kline type 99", func() error { _, e := c.Crypto.GetKline(ctx, "BTCUSDT", 99, 2, nil); return e })
	p.expectAPI("REST kline empty codes", func() error { _, e := c.Crypto.GetKline(ctx, "", KlineMin1, 2, nil); return e })
	p.expectAPI("REST kline timestamp milliseconds", func() error {
		ts := int64(1_700_000_000_000)
		_, e := c.Crypto.GetKline(ctx, "BTCUSDT", KlineMin1, 2, &ts)
		return e
	})
	p.expectAPI("REST kline timestamp year-2000", func() error {
		ts := int64(946_684_800)
		_, e := c.Crypto.GetKline(ctx, "BTCUSDT", KlineMin1, 2, &ts)
		return e
	})
	p.expectAPI("REST kline count 9999", func() error { _, e := c.Crypto.GetKline(ctx, "BTCUSDT", KlineDay, 9999, nil); return e })
	p.expectAPI("REST symbols type=US", func() error { _, e := c.Basic.GetSymbols(ctx, "US", ""); return e })
	p.expectAPI("REST symbols type=STOCK_XX", func() error { _, e := c.Basic.GetSymbols(ctx, "STOCK_XX", ""); return e })
	p.expectAPI("REST tradingDays bad date format", func() error {
		_, e := c.Basic.GetTradingDays(ctx, "US", "2026-08-01", "2026-08-15")
		return e
	})
	p.expectAPI("REST tradingDays missing beginDay", func() error {
		_, e := c.Basic.GetTradingDays(ctx, "US", "", "20260815")
		return e
	})
	p.expectAPI("REST adjustmentFactors market=STOCK_US", func() error {
		_, e := c.Basic.GetAdjustmentFactors(ctx, "AAPL.US", "STOCK_US", "20260801", "20260815")
		return e
	})
	p.expectOK("REST stockDetail type=CRYPTO → data null", func() (string, error) {
		d, err := c.Basic.GetStockDetail(ctx, SymbolCrypto, "BTCUSDT")
		if err != nil {
			return "", err
		}
		if d != nil {
			if m, ok := d.(map[string]any); ok && len(m) == 0 {
				return "server 200 data=null", nil
			}
			return "", errf("expected null/empty")
		}
		return "server 200 data=null", nil
	})
	p.expectOK("REST financial type=US uses .US suffix", func() (string, error) {
		d, err := c.Financial.GetIncomeStatement(ctx, "AAPL.US", "US", PeriodFQ)
		if err != nil {
			return "", err
		}
		arr, ok := d.([]any)
		if !ok || len(arr) == 0 {
			return "", errf("expected rows")
		}
		return "server ignored type=US and returned AAPL rows", nil
	})
	p.expectOK("REST financial period_type=xx → empty", func() (string, error) {
		d, err := c.Financial.GetIncomeStatement(ctx, "AAPL.US", SymbolStockUS, "xx")
		if err != nil {
			return "", err
		}
		arr, ok := d.([]any)
		if !ok || len(arr) > 0 {
			return "", errf("expected empty")
		}
		return "server 200 empty list", nil
	})

	badC := New(Options{APIKey: bad, MaxRetries: 1})
	defer badC.Close()
	p.expectAuth("REST bad-key crypto.trade", func() error { _, e := badC.Crypto.GetTrade(ctx, "BTCUSDT"); return e })
	p.expectAuth("REST bad-key packages.info", func() error { _, e := badC.Packages.GetInfo(ctx); return e })
	p.expectAuth("REST bad-key basic.symbols", func() error { _, e := badC.Basic.GetSymbols(ctx, SymbolCrypto, ""); return e })
	p.expectAuth("REST bad-key financial", func() error {
		_, e := badC.Financial.GetEarningStatus(ctx, "AAPL.US", SymbolStockUS)
		return e
	})

	p.wsTick(t, "WS crypto trade BTCUSDT", good, BusinessCrypto, "BTCUSDT", 25*time.Second)
	p.wsAuth(t, "WS bad-key crypto handshake", bad, BusinessCrypto, 8*time.Second)
	p.wsNews(t, "WS news handshake good key", good, 12*time.Second)
	p.wsNewsAuth(t, "WS bad-key news handshake", bad, 8*time.Second)

	var failed []string
	for _, r := range p.rows {
		line := "OK"
		if !r.ok {
			line = "FAIL"
			failed = append(failed, r.name)
		}
		t.Logf("%s  %s  (%s) %s", line, r.name, r.expect, r.detail)
	}
	if len(failed) > 0 {
		t.Fatalf("%d live cases failed: %s", len(failed), strings.Join(failed, ", "))
	}
}

func (p *probe) wsTick(t *testing.T, name, key string, biz Business, codes string, wait time.Duration) {
	t.Helper()
	ws, err := NewWebSocket(WSOptions{APIKey: key, Business: biz, MaxReconnectAttempts: 1})
	if err != nil {
		p.rows = append(p.rows, probeRow{name, "WS_TICK", err.Error(), false})
		return
	}
	ticks := make(chan struct{}, 1)
	ws.OnTrade = func(map[string]any) {
		select {
		case ticks <- struct{}{}:
		default:
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	go func() { _ = ws.Connect(ctx) }()
	ws.SubscribeTrade(codes, false)
	select {
	case <-ticks:
		p.rows = append(p.rows, probeRow{name, "WS_TICK", "tick received", true})
	case <-ctx.Done():
		p.rows = append(p.rows, probeRow{name, "WS_TICK", "no push", false})
	}
	_ = ws.Close()
}

func (p *probe) wsAuth(t *testing.T, name, key string, biz Business, wait time.Duration) {
	t.Helper()
	ws, err := NewWebSocket(WSOptions{APIKey: key, Business: biz, MaxReconnectAttempts: 1})
	if err != nil {
		p.rows = append(p.rows, probeRow{name, "InfowayAuthError", err.Error(), false})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	err = ws.Connect(ctx)
	_, ok := err.(*AuthError)
	p.rows = append(p.rows, probeRow{name, "InfowayAuthError", errString(err), ok})
	_ = ws.Close()
}

func (p *probe) wsNews(t *testing.T, name, key string, wait time.Duration) {
	t.Helper()
	news, err := NewNewsWebSocket(NewsOptions{APIKey: key, Lang: "zh-Hans", MaxReconnectAttempts: 1})
	if err != nil {
		p.rows = append(p.rows, probeRow{name, "WS_NEWS", err.Error(), false})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- news.Connect(ctx) }()
	select {
	case err := <-done:
		if _, ok := err.(*AuthError); ok {
			p.rows = append(p.rows, probeRow{name, "WS_NEWS", err.Error(), false})
		} else {
			p.rows = append(p.rows, probeRow{name, "WS_NEWS", "connected", true})
		}
	case <-ctx.Done():
		p.rows = append(p.rows, probeRow{name, "WS_NEWS", "connected", true})
	}
	_ = news.Close()
}

func (p *probe) wsNewsAuth(t *testing.T, name, key string, wait time.Duration) {
	t.Helper()
	news, err := NewNewsWebSocket(NewsOptions{APIKey: key, MaxReconnectAttempts: 1})
	if err != nil {
		p.rows = append(p.rows, probeRow{name, "InfowayAuthError", err.Error(), false})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	err = news.Connect(ctx)
	_, ok := err.(*AuthError)
	p.rows = append(p.rows, probeRow{name, "InfowayAuthError", errString(err), ok})
	_ = news.Close()
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
