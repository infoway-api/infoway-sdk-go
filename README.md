# Infoway SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/infoway-api/infoway-sdk-go.svg)](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**English** | [中文](README_CN.md)

Official Go SDK for the [Infoway](https://infoway.io) real-time financial data API. Supports stocks (HK, US, CN, JP, KS, IN, TW), crypto, and common market data via REST and WebSocket.

Walkthrough with copy-paste examples: [USAGE.md](USAGE.md) · [使用说明](USAGE_CN.md). Version **0.3.0**.

## Installation

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.3.0
```

## Quick Start

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
    ctx := context.Background()

    trades, err := client.Stock.GetTrade(ctx, "AAPL.US")
    if err != nil {
        panic(err)
    }
    fmt.Println(trades)

    klines, _ := client.Crypto.GetKline(ctx, "BTCUSDT", infoway.KlineDay, 100, nil)
    temp, _ := client.Market.GetTemperature(ctx, "HK,US", "")
    industry, _ := client.Plate.GetIndustry(ctx, "HK", 10)
    company, _ := client.StockInfo.GetCompany(ctx, "AAPL.US", "")
    pkg, _ := client.Packages.GetInfo(ctx)
    _, _, _, _ = klines, temp, industry, company
    fmt.Println(pkg)
}
```

## REST clients

| Client | Prefix | Description |
|--------|--------|-------------|
| `client.Stock` | `stock` | HK, US, CN |
| `client.Crypto` | `crypto` | Cryptocurrency |
| `client.Japan` | `japan` | Japan |
| `client.India` | `india` | India |
| `client.Korea` | `korea` | Korea (`.KS`) |
| `client.Taiwan` | `taiwan` | Taiwan (`.TW`) |
| `client.Common` | `common` | Forex, metals, futures |
| `client.Basic` | -- | Symbols, calendar, profile |
| `client.Packages` | -- | Quota for the current key |
| `client.Market` | -- | Temperature, breadth, turnover, ranks |
| `client.Plate` | -- | Industry / concept sectors |
| `client.StockInfo` | -- | Valuation, ratings, company |
| `client.Financial` | -- | Statements, dividends, earnings |

### Market data methods (stock/crypto/japan/india/korea/taiwan/common)

| Method | HTTP | Endpoint |
|--------|------|----------|
| `GetTrade(ctx, codes)` | GET | `/{prefix}/batch_trade/{codes}` |
| `GetDepth(ctx, codes)` | GET | `/{prefix}/batch_depth/{codes}` |
| `GetKline(ctx, codes, klineType, count, timestamp)` | POST | `/{prefix}/v2/batch_kline` |

## WebSocket

```go
ws, err := infoway.NewWebSocket(infoway.WSOptions{
    APIKey:   os.Getenv("INFOWAY_API_KEY"),
    Business: infoway.BusinessCrypto,
})
if err != nil {
    panic(err)
}
ws.OnTrade = func(tick map[string]any) { fmt.Println(tick["s"], tick["p"]) }
go func() { _ = ws.Connect(context.Background()) }()
ws.SubscribeTrade("BTCUSDT", false)
```

News is a separate connection (`wss://data.infoway.io/news`) and needs its own entitlement.

`OnReconnect` fires only after a later successful open. `Close()` does not fire `OnDisconnect` and wakes reconnect backoff. Unsubscribe is not replayed.

## Tests

```bash
go test ./...
INFOWAY_LIVE=1 INFOWAY_API_KEY=… go test ./... -count=1 -timeout 3m
```

## License

MIT
