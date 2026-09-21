# Infoway Go SDK usage

Go 1.22+ guide for quotes, fundamentals and live sockets. Install notes live in [README.md](README.md). Official API reference: [docs.infoway.io](https://docs.infoway.io). Chinese twin: [USAGE_CN.md](USAGE_CN.md).

- Module: `github.com/infoway-api/infoway-sdk-go` (`v0.3.0`)
- REST: `https://data.infoway.io`
- Quotes WebSocket: `wss://data.infoway.io/ws`
- News WebSocket: `wss://data.infoway.io/news`

---

## Contents

1. [Install and client](#1-install-and-client)
2. [Symbol conventions](#2-symbol-conventions)
3. [REST: market data](#3-rest-market-data)
4. [REST: basics](#4-rest-basics)
5. [REST: market overview](#5-rest-market-overview)
6. [REST: plates](#6-rest-plates)
7. [REST: stock info](#7-rest-stock-info)
8. [REST: financials](#8-rest-financials)
9. [Return values](#9-return-values)
10. [WebSocket: quotes](#10-websocket-quotes)
11. [WebSocket: news](#11-websocket-news)
12. [Errors and limits](#12-errors-and-limits)
13. [Full example](#13-full-example)
14. [Appendix: REST paths](#appendix-rest-paths)

---

## 1. Install and client

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.3.0
```

If you omit `APIKey`, both REST and WebSocket read `INFOWAY_API_KEY`. Every REST method takes `context.Context`.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    infoway "github.com/infoway-api/infoway-sdk-go"
)

func main() {
    client := infoway.New(infoway.Options{
        APIKey:     os.Getenv("INFOWAY_API_KEY"),
        BaseURL:    "https://data.infoway.io", // optional
        Timeout:    15 * time.Second,          // default 15s
        MaxRetries: 3,                         // default 3, exponential backoff
    })
    defer client.Close()

    data, err := client.Crypto.GetTrade(context.Background(), "BTCUSDT")
    if err != nil {
        panic(err)
    }
    fmt.Println(data)
}
```

| Option | Default | Meaning |
|--------|---------|---------|
| `APIKey` | `INFOWAY_API_KEY` | API key |
| `BaseURL` | `https://data.infoway.io` | REST root |
| `Timeout` | `15s` | Per-request timeout |
| `MaxRetries` | `3` | Retries |

| Entry | Use |
|-------|-----|
| `Stock` / `Crypto` / `Japan` / `India` / `Korea` / `Taiwan` / `Common` | Trade, depth, kline |
| `Basic` | Symbols, calendar, single-name profile |
| `Packages` | Quota for the current key |
| `Market` | Sentiment, breadth, turnover, ranks |
| `Plate` | Industry / concept sectors |
| `StockInfo` | Valuation, ratings, company |
| `Financial` | Statements, dividends, earnings |

| Const | Wire values | Used by |
|-------|-------------|---------|
| `KlineMin1`…`KlineYear` | 1–12 | K-line REST / WS |
| `SymbolStockUS` / `SymbolCrypto`… | `STOCK_US` / `CRYPTO`… | `Basic` / `Financial` |
| `MarketHK`… | `HK` `US` `CN` `JP` `KS` `TW` `IN` | Overview, plates, calendar |
| `LangEN` / `LangZHCN` | `en` / `zh-CN` | REST `lang` |
| `NewsEN` / `NewsZHHans`… | `en` / `zh-Hans`… | News WS |
| `PeriodFQ` / `PeriodFY` / `PeriodFH` | `fq` / `fy` / `fh` | Financials |
| `BusinessCrypto` / `BusinessKorea`… | quote WS `business` | Quotes WS |
| `ScheduleEnergy`… | `ENERGY` `FOREX` `FUTURES` `METAL` `INDICES` | Trading-schedule filter |

---

## 2. Symbol conventions

| Market | Form | Good | Bad |
|--------|------|------|-----|
| US | `.US` | `AAPL.US` | `AAPL` |
| HK | `.HK`, 5-digit pad | `00700.HK` | `700.HK` |
| Shanghai | `.SH` | `600519.SH` | `600519.CN` |
| Shenzhen | `.SZ` | `000001.SZ` | `000001.CN` |
| Japan | `.JP` | `7203.JP` | |
| Korea | `.KS` | `005930.KS` | |
| India | `.IN` | `RELIANCE.IN` | |
| Taiwan | `.TW` | `2330.TW` | |
| Crypto | pair | `BTCUSDT` | |
| FX / metals | pair | `USDJPY` | |

`Basic` / `Financial` / `GetStockDetail` take a **product type**, not a market code `US`:

`STOCK_US` `STOCK_CN` `STOCK_HK` `STOCK_JP` `STOCK_KS` `STOCK_IN` `STOCK_TW`  
`CRYPTO` `FOREX` `FUTURES` `ENERGY` `METAL` `INDICES`

`GetSymbols` with `market=US` returns HTTP 400 `Required parameter 'type' is not present.`

---

## 3. REST: market data

Seven market clients share the same methods; only the path prefix changes. Join codes with commas.

```go
ctx := context.Background()
us, _ := client.Stock.GetTrade(ctx, "AAPL.US,TSLA.US")
btc, _ := client.Crypto.GetTrade(ctx, "BTCUSDT")
kr, _ := client.Korea.GetTrade(ctx, "005930.KS")
tw, _ := client.Taiwan.GetTrade(ctx, "2330.TW")

tick := btc.([]any)[0].(map[string]any)
fmt.Println(tick["s"], tick["p"])

_, _ = client.Crypto.GetDepth(ctx, "BTCUSDT")
latest, _ := client.Crypto.GetKline(ctx, "BTCUSDT", infoway.KlineMin1, 100, nil)
historical, _ := client.Crypto.GetKline(ctx, "BTCUSDT", infoway.KlineMin1, 100, infoway.Ptr(int64(1_700_000_000)))
_, _, _ = us, kr, tw
_, _ = latest, historical
```

Trade fields: `s` symbol, `p` price, `v` size, `vw` turnover, `t` milliseconds, `td` side (0 default / 1 buy / 2 sell).  
K-lines are wrapped per symbol; bars sit in `respList`; `t` is a **seconds** string. `timestamp` only applies to minute / hour bars.

At most **500** bars per symbol. Multi-symbol kline requests are capped at 2 bars each. The SDK always uses `POST /{market}/v2/batch_kline`.

---

## 4. REST: basics

Dates are always `YYYYMMDD`.

```go
ctx := context.Background()
_, _ = client.Basic.GetSymbols(ctx, infoway.SymbolStockUS, "")
_, _ = client.Basic.GetSymbols(ctx, infoway.SymbolStockTW, "2330.TW")
_, _ = client.Basic.GetSymbolInfo(ctx, infoway.SymbolStockUS, "AAPL.US")
_, _ = client.Basic.GetStockDetail(ctx, infoway.SymbolStockUS, "AAPL.US")
_, _ = client.Basic.GetAdjustmentFactors(ctx, "AAPL.US", "US", "20260801", "20260815")
_, _ = client.Basic.GetTradingDays(ctx, "US", "20260801", "20260815")
_, _ = client.Basic.GetTradingSchedule(ctx)
_, _ = client.Basic.GetTradingScheduleByType(ctx, "ENERGY")
_, _ = client.Basic.GetMarkets(ctx)
fmt.Println(client.Packages.GetInfo(ctx))
```

`GetTradingHours` is a deprecated alias of `GetTradingSchedule`. The schedule endpoint has **no market filter**. Filter by product with `ENERGY` / `FOREX` / `FUTURES` / `METAL` / `INDICES`. Passing `STOCK_US` returns an error before the request.

`Packages.GetInfo()` is `GET /package/info`: `packageName`, `expireTime`, `apiNumPerSec`, `maxWsConNum`, `maxNum`, `maxYearHisData`, `allWsNum`.

---

## 5. REST: market overview

```go
ctx := context.Background()
_, _ = client.Market.GetTemperature(ctx, infoway.JoinMarkets(infoway.MarketHK, infoway.MarketUS), string(infoway.LangZHCN))
_, _ = client.Market.GetBreadth(ctx, "US", "zh-CN")
_, _ = client.Market.GetTurnover(ctx, "US", "")
_, _ = client.Market.GetIndexes(ctx, "en")
_, _ = client.Market.GetLeaders(ctx, "US", 10, "")
_, _ = client.Market.GetOverview(ctx, "US", "zh-CN")
_, _ = client.Market.GetRankCategories(ctx, "US", "")
_, _ = client.Market.GetRank(ctx, "US", "all", &infoway.RankOptions{
    Sort: "chg", Order: "desc", Limit: 30, Lang: "en",
})
```

Rank `key` values come from `GetRankCategories`. `GetRankConfig` is deprecated (HTTP 404).

---

## 6. REST: plates

```go
ctx := context.Background()
_, _ = client.Plate.GetIndustry(ctx, "HK", 200)
_, _ = client.Plate.GetConcept(ctx, "HK", 100)
_, _ = client.Plate.GetMembers(ctx, "IN20293.HK", 0, 50)
_, _ = client.Plate.GetIntro(ctx, "IN20293.HK")
_, _ = client.Plate.GetChart(ctx, "HK", 50)
```

---

## 7. REST: stock info

Optional `lang`: `en` or `zh-CN`. Pass `""` to omit.

```go
ctx := context.Background()
symbol := "AAPL.US"
_, _ = client.StockInfo.GetValuation(ctx, symbol, "")
_, _ = client.StockInfo.GetRatings(ctx, symbol, "")
_, _ = client.StockInfo.GetCompany(ctx, symbol, "zh-CN")
_, _ = client.StockInfo.GetPanorama(ctx, symbol, "")
_, _ = client.StockInfo.GetConcepts(ctx, symbol, "")
_, _ = client.StockInfo.GetEvents(ctx, symbol, 20, "")
_, _ = client.StockInfo.GetDrivers(ctx, symbol, "")
```

---

## 8. REST: financials

Every method needs `symbol` plus `type`. Statement-style methods accept `period_type` (`fq` / `fy` / `fh`). Empty period omits the query.

```go
ctx := context.Background()
symbol := "AAPL.US"
typ := infoway.SymbolStockUS
_, _ = client.Financial.GetEarningStatus(ctx, symbol, typ)
_, _ = client.Financial.GetIncomeStatement(ctx, symbol, typ, infoway.PeriodFQ)
_, _ = client.Financial.GetRevenue(ctx, symbol, typ, "")
_, _ = client.Financial.GetCashFlow(ctx, symbol, typ, infoway.PeriodFY)
_, _ = client.Financial.GetBalanceSheet(ctx, symbol, typ, "")
_, _ = client.Financial.GetStatistics(ctx, symbol, typ, "")
_, _ = client.Financial.GetDividend(ctx, symbol, typ, "")
_, _ = client.Financial.GetDividendPayout(ctx, symbol, typ)
_, _ = client.Financial.GetEarnings(ctx, symbol, typ, infoway.PeriodFQ)
```

HK example: `client.Financial.GetDividend(ctx, "00700.HK", infoway.SymbolStockHK, "")`.

Production is lenient on a few financial filters: `type=US` still returns rows for `AAPL.US` (suffix wins); `period_type=xx` returns HTTP 200 and an empty list. `GetStockDetail(..., CRYPTO)` returns `data: null` with HTTP 200.

---

## 9. Return values

REST methods return decoded JSON as `any` (`[]any` or `map[string]any`). Numbers become `float64`, prices stay strings on the wire (`"305.771"`). Trade `t` is milliseconds; kline `t` is a seconds string; `vw` is turnover, not VWAP. Depth `a`/`b` are `[[prices…],[qtys…]]`.

---

## 10. WebSocket: quotes

Separate from the REST client. `Business` must match the market; a wrong channel acks and then pushes nothing.

`PrintFrames` defaults to **false**. REST and WebSocket both fall back to `INFOWAY_API_KEY`. `Connect` blocks until the context is cancelled, `Close` is called, or a 401 handshake.

```go
ws, err := infoway.NewWebSocket(infoway.WSOptions{
    APIKey:   os.Getenv("INFOWAY_API_KEY"),
    Business: infoway.BusinessCrypto, // stock / japan / india / korea / taiwan / crypto / common
    PrintFrames: false,
})
if err != nil {
    panic(err)
}
ws.OnTrade = func(data map[string]any) { fmt.Println("TRADE", data["s"], data["p"]) }
ws.OnDepth = func(data map[string]any) { fmt.Println("DEPTH", data["s"]) }
ws.OnKline = func(data map[string]any) { fmt.Println("KLINE", data["s"], data["ty"]) }
ws.OnError = func(err error) { fmt.Fprintln(os.Stderr, err) }

ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() { _ = ws.Connect(ctx) }()
ws.SubscribeTrade("BTCUSDT,ETHUSDT", false)
ws.SubscribeDepth("BTCUSDT,ETHUSDT")
ws.SubscribeKline("BTCUSDT", infoway.KlineMin1)
time.Sleep(30 * time.Second)
ws.UnsubscribeKline("BTCUSDT", infoway.KlineMin1)
_ = ws.Close()
```

Equity trade types need `includeTy=true`. Crypto still omits `ty`. Callbacks receive **`data`**, not the envelope. Subscribe before `Connect`; the client replays on open.

Lifecycle (reconnect / subscribe / close):

- Heartbeat `10010` every 30s, **one goroutine per live session**. A drop schedules **at most one** reconnect (1s → 30s backoff). After a live session the backoff resets.
- `OnReconnect` fires only after a later successful open, never on the first `Connect`.
- `OnDisconnect` fires on an unexpected drop. `Close()` does **not** fire it and does **not** reconnect; it also wakes a pending backoff so Close returns immediately.
- The client keeps the **desired** subscription set. Unsubscribe is forgotten and is **not** replayed. Subscribe while reconnecting is flushed on the next open. Unsubscribing one kline interval leaves the others.
- HTTP 401 still stops reconnecting.

| Symptom | Cause |
|---------|-------|
| First frame is plain text `You have permission...` | `business=stock` greeting; SDK skips it |
| `{"code":200,"msg":"ws connect success"}` | Welcome, not an error |
| ack `ok` then silence | Wrong business, unknown code, or closed market |
| No heartbeat reply | Server does not answer `10010`; do not reconnect on missing ack |
| Dropped after many frames | **60 frames/minute/connection**; merge codes |
| HTTP 401 | Bad key; `*AuthError`, no reconnect |

Protocol:

| Dir | Code | Meaning |
|-----|------|---------|
| out | 10000 / 10003 / 10006 | Subscribe trade / depth / kline |
| out | 11000 / 11001 / 11002 | Unsubscribe |
| out | 10010 | Heartbeat (30s; `ack=1` yields 10011) |
| in | 10001 / 10004 / 10007 | Subscribe ack |
| in | 10002 / 10005 / 10008 | Push |
| in | 10011 | Heartbeat ack (optional) |
| in | 11010 | Unsubscribe ack (quotes + news) |
| in | 200 | Welcome |
| in | 500–521 | Server error → `OnError` (501/502 rate limit; see `WsErrorCode`) |

508 on WebSocket is `APIKEY_EXPIRED`, not REST `PRODUCT_NOT_EXISTS`. Do not decode REST `ret` with `WsErrorCode`.

---

## 11. WebSocket: news

`wss://data.infoway.io/news`, separate entitlement. One news connection per key. Default language is `en`.

```go
news, err := infoway.NewNewsWebSocket(infoway.NewsOptions{
    APIKey: os.Getenv("INFOWAY_API_KEY"),
    Lang:   string(infoway.NewsZHHans),
})
if err != nil {
    panic(err)
}
news.OnNews = func(item map[string]any) { fmt.Println(item["title"], item["sd"]) }
news.OnNewsParsed = func(item map[string]any) { fmt.Println(item["published"]) } // time.Time
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() { _ = news.Connect(ctx) }()
time.Sleep(time.Minute)
news.Unsubscribe()          // 11020
news.Subscribe("en")        // replaces language
_ = news.Close()
```

Push fields: `dk` (dedup), `country`, `lang`, `route`, `title`, `published` (**seconds**), `urgency` (lower = hotter), `provider`, `symbols[]`, `link`, `content`, `sd` (summary).

| Dir | Code | Meaning |
|-----|------|---------|
| out | 10020 / 11020 | Subscribe / unsubscribe |
| in | 10021 / 10022 | Ack / push |

A key without news access fails the handshake with HTTP 401. The current `Lang` is replayed on reconnect; `Unsubscribe()` clears it so reconnect will not resubscribe. `OnReconnect` / `OnDisconnect` follow the same rules as the quotes socket.

---

## 12. Errors and limits

```go
_, err := client.Stock.GetTrade(ctx, "INVALID")
switch e := err.(type) {
case *infoway.AuthError:
    fmt.Fprintln(os.Stderr, "auth:", e.Msg)
case *infoway.RateLimitError:
    fmt.Fprintf(os.Stderr, "rate [%d %s] %s\n", e.Ret, e.ErrorName, e.Msg)
case *infoway.TimeoutError:
    fmt.Fprintln(os.Stderr, "timeout:", e)
case *infoway.IOError:
    fmt.Fprintln(os.Stderr, "io:", e)
case *infoway.APIError:
    // HTTP → RestErrorCode; WS → WsErrorCode. 508–514 collide.
    fmt.Fprintf(os.Stderr, "ret=%d name=%s msg=%s trace=%s\n", e.Ret, e.ErrorName, e.Msg, e.TraceID)
}
```

Envelope handling matches the other official SDKs: HTTP 401 / 429 first; `{detail}`-only bodies (rate limit may be HTTP 200); RFC 7807; business `ret`/`code` **before** HTTP ≥ 400; then gateway pages. 501/502 retry as rate limits.

`Error()` includes the code and enum name, e.g. REST `[508 PRODUCT_NOT_EXISTS] All product not exists` vs WebSocket `[508 APIKEY_EXPIRED] …`. Do not decode REST `ret` with `WsErrorCode`.

A forged key is `*AuthError` `[401] Token invalid` on REST, and handshake HTTP 401 (no reconnect) on both sockets.

### REST `ret` (`RestErrorCode`)

| Code | Name | Meaning |
|------|------|---------|
| 200 | SUCCESS | OK |
| 400 | BAD_REQUEST | commonApi bad params (HTTP 400 too) |
| 500 | SERVER_ERROR | Uncaught error, **or** production quote errors that still use 500 + the enum text |
| 501 / 502 | REQUEST_EXCEED_LIMIT / REQUEST_FOR_DAY_LIMIT | Rate limit → `*RateLimitError` |
| 503 | KLINE_EXCEEDS_LIMIT | Too many bars |
| 505 | PRODUCTS_EXCEEDS_LIMIT | Too many symbols |
| 506 / 507 | PARAM_ERROR / PARAM_LOST | Bad / missing field |
| **508** | **PRODUCT_NOT_EXISTS** | **Unknown product** (not WS key expiry) |
| 509 | TOKEN_PERMISSION_EXPIRED | Token permission expired |
| 513 | TIME_LIMIT_ERROR | K-line `timestamp` older than the package |
| **514** | **NO_PERMISSION** | **No market entitlement** (not WS URL error) |

`/common/basic/*` only uses 200 / 400 / 500.

Verified against production (2026-09-21): quote-service business errors still arrive as **HTTP 200 + `ret=500`** with the English template (`All product not exists`, `Param error：klineType`, `Timestamp limit error…`, `Kline quantity exceeds the limit：500`). The SDK surfaces the **message**.

### WebSocket `code` (`WsErrorCode`)

| Code | Name | Meaning |
|------|------|---------|
| 501 / 502 | REQUEST_FREQUENCY_MIN_EXCEED / DAY | 60 frames/min → `*RateLimitError` |
| 505 / 516 | PRODUCTS_QUANTITY_EXCEED | Per connection / all connections |
| 506 / 507 | PARAM_ERROR / PARAM_LOST | Bad or missing fields |
| **508–511** | **APIKEY_*** | **Expired / invalid / empty / blacklist** |
| 512 / 513 / 514 | Conn cap / heartbeat timeout / bad URL | 513 then close |
| 515 | PARAM_NOT_JSON | Inbound text was not JSON |
| 517–521 | Handshake | Missing key / no entitlement; 519/520 differ on Korea, Taiwan, news |

REST budget is about **1200 calls/minute/key**. HTTP 429, REST `ret` 501/502, or `{"detail":"Rate limit exceeded"}` are retried with backoff.

---

## 13. Full example

Pull a REST snapshot, then hang a live trade.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    infoway "github.com/infoway-api/infoway-sdk-go"
)

func main() {
    apiKey := os.Getenv("INFOWAY_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "Set INFOWAY_API_KEY")
        os.Exit(1)
    }

    client := infoway.New(infoway.Options{APIKey: apiKey})
    defer client.Close()
    ctx := context.Background()

    trade, err := client.Crypto.GetTrade(ctx, "BTCUSDT")
    if err != nil {
        panic(err)
    }
    fmt.Println("REST last:", trade)

    company, err := client.StockInfo.GetCompany(ctx, "AAPL.US", "zh-CN")
    if err != nil {
        panic(err)
    }
    fmt.Println("company:", company)

    earn, err := client.Financial.GetEarningStatus(ctx, "AAPL.US", infoway.SymbolStockUS)
    if err != nil {
        panic(err)
    }
    fmt.Println("earn:", earn)

    pkg, err := client.Packages.GetInfo(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println("quota:", pkg)

    ws, err := infoway.NewWebSocket(infoway.WSOptions{
        APIKey:   apiKey,
        Business: infoway.BusinessCrypto,
    })
    if err != nil {
        panic(err)
    }
    first := make(chan struct{}, 1)
    ws.OnTrade = func(data map[string]any) {
        fmt.Println("WS trade:", data)
        select {
        case first <- struct{}{}:
        default:
        }
    }
    ws.OnError = func(err error) { fmt.Fprintln(os.Stderr, err) }

    wsCtx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go func() { _ = ws.Connect(wsCtx) }()
    ws.SubscribeTrade("BTCUSDT,ETHUSDT", false)

    select {
    case <-first:
    case <-time.After(45 * time.Second):
        fmt.Fprintln(os.Stderr, "no trade push in 45s")
    }
    _ = ws.Close()
}
```

```bash
export INFOWAY_API_KEY=your-key
go run .
```

Live contract / deep probe (opt-in):

```bash
cd sdks/go
INFOWAY_LIVE=1 INFOWAY_API_KEY=your-key go test ./... -count=1 -timeout 3m
```

---

## Appendix: REST paths

Quotes (`{market}` = `stock` / `crypto` / `japan` / `india` / `korea` / `taiwan` / `common`):

- `GET /{market}/batch_trade/{codes}`
- `GET /{market}/batch_depth/{codes}`
- `POST /{market}/v2/batch_kline`

Basics / financials / quota:

- `GET /common/basic/symbols`
- `GET /common/basic/symbols/info`
- `GET /common/basic/symbols/adjustment_factors`
- `GET /common/basic/markets/trading_days`
- `GET /common/basic/markets/trading_schedule`
- `GET /common/basic/markets`
- `GET /common/basic/stock/detail`
- `GET /common/basic/financial/{earning_status|income_statement|revenue|cash_flow|balance_sheet|statistics|dividend|dividend_payout|earnings}`
- `GET /package/info`

Market / plate / stock:

- `GET /common/v2/basic/market/{temperature|indexes}`
- `GET /common/v2/basic/market/{breadth|turnover|leaders|overview|rank/categories}/{market}`
- `GET /common/v2/basic/market/rank/{market}/{key}`
- `GET /common/v2/basic/plate/{industry|concept|chart}/{market}`
- `GET /common/v2/basic/plate/{members|intro}/{plateSymbol}`
- `GET /common/v2/basic/stock/{valuation|ratings|company|panorama|concepts|events|drivers}/{symbol}`
