package infoway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBasicPaths(t *testing.T) {
	var path string
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = io.WriteString(w, `{"ret":200,"data":[]}`)
	})
	ctx := context.Background()
	_, _ = c.Basic.GetSymbols(ctx, SymbolStockUS, "AAPL.US")
	if path != "/common/basic/symbols" {
		t.Fatal(path)
	}
	_, _ = c.Basic.GetStockDetail(ctx, SymbolStockUS, "AAPL.US")
	if path != "/common/basic/stock/detail" {
		t.Fatal(path)
	}
	_, _ = c.Basic.GetMarkets(ctx)
	if path != "/common/basic/markets" {
		t.Fatal(path)
	}
	_, _ = c.Packages.GetInfo(ctx)
	if path != "/package/info" {
		t.Fatal(path)
	}
}

func TestScheduleRejectsEquity(t *testing.T) {
	c := New(Options{APIKey: "k", BaseURL: "http://127.0.0.1:1"})
	_, err := c.Basic.GetTradingScheduleByType(context.Background(), "STOCK_US")
	if err == nil || !strings.Contains(err.Error(), "ENERGY/FOREX") {
		t.Fatalf("%v", err)
	}
	_, err = c.Basic.GetTradingScheduleByType(context.Background(), string(SymbolStockUS))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleEnergyPath(t *testing.T) {
	var rawQuery string
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"ret":200,"data":[]}`)
	})
	_, err := c.Basic.GetTradingScheduleByType(context.Background(), "ENERGY")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rawQuery, "type=ENERGY") {
		t.Fatal(rawQuery)
	}
}

func TestFinancialAndRankPaths(t *testing.T) {
	var path, q string
	c, _ := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		path, q = r.URL.Path, r.URL.RawQuery
		_, _ = io.WriteString(w, `{"ret":200,"data":[]}`)
	})
	ctx := context.Background()
	_, _ = c.Financial.GetIncomeStatement(ctx, "AAPL.US", SymbolStockUS, PeriodFQ)
	if path != "/common/basic/financial/income_statement" || !strings.Contains(q, "period_type=fq") {
		t.Fatalf("%s %s", path, q)
	}
	_, _ = c.Financial.GetEarningStatus(ctx, "AAPL.US", SymbolStockUS)
	if path != "/common/basic/financial/earning_status" {
		t.Fatal(path)
	}
	_, _ = c.Market.GetRank(ctx, "US", "all", &RankOptions{Sort: "chg", Order: "desc", Limit: 5, Lang: "en"})
	if !strings.Contains(path, "/common/v2/basic/market/rank/US/all") || !strings.Contains(q, "sort=chg") {
		t.Fatalf("%s %s", path, q)
	}
	_, _ = c.Korea.GetTrade(ctx, "005930.KS")
	if path != "/korea/batch_trade/005930.KS" {
		t.Fatal(path)
	}
	_, _ = c.Taiwan.GetTrade(ctx, "2330.TW")
	if path != "/taiwan/batch_trade/2330.TW" {
		t.Fatal(path)
	}
}

func TestOfRestVsOfWS508(t *testing.T) {
	r := OfRest(508, "", "")
	if r.ErrorName != "PRODUCT_NOT_EXISTS" || r.Msg != "All product not exists" {
		t.Fatalf("%+v", r)
	}
	w := OfWS(508, "", "")
	if w.ErrorName != "APIKEY_EXPIRED" || w.Msg != "API key expired" {
		t.Fatalf("%+v", w)
	}
}
