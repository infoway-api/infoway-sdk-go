# Infoway Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/infoway-api/infoway-sdk-go.svg)](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**English** | [中文](README_CN.md)

Official Infoway Go SDK for REST market data, fundamentals, and WebSocket streams.

| Item | Description |
| --- | --- |
| Module | [`github.com/infoway-api/infoway-sdk-go@v0.4.0`](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go) |
| Runtime | Go 1.22+ |
| REST | `https://data.infoway.io` |
| Quotes WebSocket | `wss://data.infoway.io/ws` |
| News WebSocket | `wss://data.infoway.io/news` |
| Rate limits | [REST](https://docs.infoway.io/en-docs/getting-started/api-limitation/rest-api-limitation) · [WebSocket](https://docs.infoway.io/en-docs/getting-started/api-limitation/websocket-limitation) |
| Error codes | [REST](https://docs.infoway.io/en-docs/getting-started/error-codes/rest-api-error-codes) · [WebSocket](https://docs.infoway.io/en-docs/getting-started/error-codes/websocket-error-codes) |
| Endpoints | [Endpoints](https://docs.infoway.io/en-docs/getting-started/endpoints) |

If `APIKey` is omitted, the SDK reads `INFOWAY_API_KEY`. REST methods take `context.Context` and return errors as values.

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [Symbols](#symbols)
- [Client](#client)
- [REST](#rest)
- [Return values](#return-values)
- [WebSocket](#websocket)
- [Error codes](#error-codes)
- [REST paths](#rest-paths)

## Install

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.4.0
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    infoway "github.com/infoway-api/infoway-sdk-go"
)

func main() {
    client := infoway.New(infoway.Options{APIKey: os.Getenv("INFOWAY_API_KEY")})
    defer client.Close()

    trades, err := client.Stock.GetTrade(context.Background(), "AAPL.US")
    if err != nil {
        panic(err)
    }
    fmt.Println(trades)
}
```

## Symbols

| Market | Format | Valid | Invalid |
| --- | --- | --- | --- |
| US | `{code}.US` | `AAPL.US` | `AAPL` |
| Hong Kong | 5-digit + `.HK` | `00700.HK` | `700.HK` |
| Shanghai | `{code}.SH` | `600519.SH` | `600519.CN` |
| Shenzhen | `{code}.SZ` | `000001.SZ` | `000001.CN` |
| Japan | `{code}.JP` | `7203.JP` | |
| Korea | `{code}.KS` | `005930.KS` | |
| India | `{code}.IN` | `RELIANCE.IN` | |
| Taiwan | `{code}.TW` | `2330.TW` | |
| Crypto | Pair | `BTCUSDT` | |
| FX | Pair | `USDJPY` | |

`Basic` / `Financial` take a product `type` such as `STOCK_US` or `CRYPTO`, not a market code like `US`.

## Client

| Option | Default | Description |
| --- | --- | --- |
| `APIKey` | `INFOWAY_API_KEY` | API key |
| `BaseURL` | `https://data.infoway.io` | REST base URL |
| `Timeout` | `15s` | Per-request timeout |
| `MaxRetries` | `3` | Retry count |

Clients: `Stock` / `Crypto` / `Japan` / `India` / `Korea` / `Taiwan` / `Common` / `Basic` / `Packages` / `Market` / `Plate` / `StockInfo` / `Financial`.

## REST

Join symbols with commas. Rate limits: [REST API Limitation](https://docs.infoway.io/en-docs/getting-started/api-limitation/rest-api-limitation).

### Quotes

| Method | Description |
| --- | --- |
| `GetTrade(ctx, codes)` | Latest trade |
| `GetDepth(ctx, codes)` | Order book |
| `GetKline(ctx, codes, klineType, count, timestamp)` | Candles. Pass `nil` for the latest |

```go
ctx := context.Background()
client.Stock.GetTrade(ctx, "AAPL.US,TSLA.US")
client.Crypto.GetDepth(ctx, "BTCUSDT")
client.Korea.GetTrade(ctx, "005930.KS")
client.Taiwan.GetTrade(ctx, "2330.TW")
client.Crypto.GetKline(ctx, "BTCUSDT", infoway.KlineMin1, 100, nil)
```

| Constant | Value | Interval |
| --- | --- | --- |
| `KlineMin1` / `KlineMin5` / `KlineMin15` / `KlineMin30` | 1–4 | Minutes |
| `KlineHour1` / `KlineHour2` / `KlineHour4` | 5–7 | Hours |
| `KlineDay` / `KlineWeek` / `KlineMonth` / `KlineQuarter` / `KlineYear` | 8–12 | Daily and above |

Up to 500 bars per symbol; multi-symbol calls return 2 bars each. `vw` is turnover.

### Basic info

Dates use `YYYYMMDD`.

```go
client.Basic.GetSymbols(ctx, infoway.SymbolStockUS, "")
client.Basic.GetSymbolInfo(ctx, infoway.SymbolStockUS, "AAPL.US")
client.Basic.GetStockDetail(ctx, infoway.SymbolStockUS, "AAPL.US")
client.Basic.GetAdjustmentFactors(ctx, "AAPL.US", "US", "20260801", "20260815")
client.Basic.GetTradingDays(ctx, "US", "20260801", "20260815")
client.Basic.GetTradingSchedule(ctx)
client.Basic.GetTradingScheduleByType(ctx, "ENERGY")
client.Basic.GetMarkets(ctx)
client.Packages.GetInfo(ctx)
```

### Overview / sectors / stock info / financials

```go
client.Market.GetTemperature(ctx, infoway.JoinMarkets(infoway.MarketHK, infoway.MarketUS), "zh-CN")
client.Market.GetBreadth(ctx, "US", "zh-CN")
client.Market.GetTurnover(ctx, "US", "")
client.Market.GetIndexes(ctx, "en")
client.Market.GetLeaders(ctx, "US", 10, "")
client.Market.GetOverview(ctx, "US", "zh-CN")
client.Market.GetRankCategories(ctx, "US", "")
client.Market.GetRank(ctx, "US", "all", &infoway.RankOptions{
    Sort: "chg", Order: "desc", Limit: 30, Lang: "en",
})

client.Plate.GetIndustry(ctx, "HK", 200)
client.Plate.GetConcept(ctx, "HK", 100)
client.Plate.GetMembers(ctx, "IN20293.HK", 0, 50)
client.Plate.GetIntro(ctx, "IN20293.HK")
client.Plate.GetChart(ctx, "HK", 50)

client.StockInfo.GetValuation(ctx, "AAPL.US", "")
client.StockInfo.GetRatings(ctx, "AAPL.US", "")
client.StockInfo.GetCompany(ctx, "AAPL.US", "zh-CN")
client.StockInfo.GetPanorama(ctx, "AAPL.US", "")
client.StockInfo.GetConcepts(ctx, "AAPL.US", "")
client.StockInfo.GetEvents(ctx, "AAPL.US", 20, "")
client.StockInfo.GetDrivers(ctx, "AAPL.US", "")

typ := infoway.SymbolStockUS
client.Financial.GetEarningStatus(ctx, "AAPL.US", typ)
client.Financial.GetIncomeStatement(ctx, "AAPL.US", typ, infoway.PeriodFQ)
client.Financial.GetRevenue(ctx, "AAPL.US", typ, "")
client.Financial.GetCashFlow(ctx, "AAPL.US", typ, infoway.PeriodFY)
client.Financial.GetBalanceSheet(ctx, "AAPL.US", typ, "")
client.Financial.GetStatistics(ctx, "AAPL.US", typ, "")
client.Financial.GetDividend(ctx, "00700.HK", infoway.SymbolStockHK, "")
client.Financial.GetDividendPayout(ctx, "AAPL.US", typ)
client.Financial.GetEarnings(ctx, "AAPL.US", typ, infoway.PeriodFQ)
```

Rank `key` values come from `GetRankCategories`. Financial methods require `symbol` and `type`. `period_type`: `fq` quarter, `fy` year, `fh` half-year.

## Return values

REST methods decode JSON into `any` (`[]any` or `map[string]any`). Numbers are `float64`; prices remain strings on the wire. Trade `t` is milliseconds; candle `t` is a seconds string; book `a`/`b` is `[[price…],[qty…]]`.

## WebSocket

### Quotes

`Business` must match the symbol market. `Connect` blocks — run it in a goroutine. 60 frames per minute per connection. See [WebSocket Limitation](https://docs.infoway.io/en-docs/getting-started/api-limitation/websocket-limitation).

```go
ws, err := infoway.NewWebSocket(infoway.WSOptions{
    APIKey:   os.Getenv("INFOWAY_API_KEY"),
    Business: infoway.BusinessCrypto,
})
if err != nil {
    panic(err)
}
ws.OnTrade = func(data map[string]any) { fmt.Println(data["s"], data["p"]) }
ws.OnError = func(err error) { fmt.Fprintln(os.Stderr, err) }

ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() { _ = ws.Connect(ctx) }()
ws.SubscribeTrade("BTCUSDT,ETHUSDT", false)
ws.SubscribeKline("BTCUSDT", infoway.KlineMin1)
ws.UnsubscribeKline("BTCUSDT", infoway.KlineMin1)
_ = ws.Close()
```

Equity trade types: `SubscribeTrade(codes, true)`.

| Behavior | Description |
| --- | --- |
| Heartbeat | `10010` every 30 seconds; server does not reply |
| Reconnect | Replays the current subscription set |
| `OnReconnect` | After a later successful open |
| `OnDisconnect` | Unexpected drop only. `Close()` does not fire it |
| HTTP 401 | Stops reconnecting |

| Dir | Code | Description |
| --- | --- | --- |
| out | 10000 / 10003 / 10006 | Subscribe trade / depth / kline |
| out | 11000 / 11001 / 11002 | Unsubscribe |
| out | 10010 | Heartbeat |
| in | 10002 / 10005 / 10008 | Push |
| in | 11010 | Unsubscribe ack |
| in | 200 | Connected |

### News

`wss://data.infoway.io/news` requires a separate entitlement. One news connection per key.

```go
news, err := infoway.NewNewsWebSocket(infoway.NewsOptions{
    APIKey: os.Getenv("INFOWAY_API_KEY"),
    Lang:   string(infoway.NewsZHHans),
})
if err != nil {
    panic(err)
}
news.OnNews = func(item map[string]any) { fmt.Println(item["title"]) }
```

Subscribe `10020`, unsubscribe `11020`, push `10022`.

## Error codes

REST uses `ret`. WebSocket uses `code`. `508`–`514` mean different things on each side. Full tables: [REST API Error Codes](https://docs.infoway.io/en-docs/getting-started/error-codes/rest-api-error-codes) and [WebSocket Error Codes](https://docs.infoway.io/en-docs/getting-started/error-codes/websocket-error-codes).

```go
_, err := client.Stock.GetTrade(ctx, "INVALID")
switch e := err.(type) {
case *infoway.AuthError:
    fmt.Println(e.Msg)
case *infoway.RateLimitError:
    fmt.Println(e.Ret, e.Msg)
case *infoway.APIError:
    fmt.Println(e.Error(), e.TraceID)
}
```

REST budget is about 1200 calls/minute/key. An invalid key is `*AuthError` on REST; WebSocket handshake HTTP 401 does not reconnect.

| REST `ret` | Description |
| --- | --- |
| 200 | Success |
| 400 | Bad request |
| 500 | Server error |
| 501 / 502 | Rate limit |
| 503 | Candle count exceeded |
| 505 | Symbol count exceeded |
| 506 / 507 | Invalid / missing parameter |
| 508 | Symbol not found |
| 509 | Permission expired |
| 513 | Timestamp outside plan history |
| 514 | No permission |

| WebSocket `code` | Description |
| --- | --- |
| 501 / 502 | Rate limit |
| 505 / 516 | Subscription count exceeded |
| 506 / 507 | Invalid / missing parameter |
| 508–511 | API key expired / invalid / empty / blacklisted |
| 512 | Connection count exceeded |
| 513 | Heartbeat timeout |
| 515 | Not JSON |
| 517–521 | Handshake failed |

## REST paths

`{market}` = `stock` / `crypto` / `japan` / `india` / `korea` / `taiwan` / `common`. Full list: [Endpoints](https://docs.infoway.io/en-docs/getting-started/endpoints).

| API | Path |
| --- | --- |
| Latest trade | `GET /{market}/batch_trade/{codes}` |
| Order book | `GET /{market}/batch_depth/{codes}` |
| Candles | `POST /{market}/v2/batch_kline` |
| Symbols / calendar / financials | `GET /common/basic/*` |
| Overview / sectors / stock info | `GET /common/v2/basic/*` |
| Package | `GET /package/info` |

## License

MIT. API key: [infoway.io](https://infoway.io).
