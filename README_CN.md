# Infoway Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/infoway-api/infoway-sdk-go.svg)](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[English](README.md) | **中文**

[Infoway](https://infoway.io) 实时金融数据 API 官方 Go SDK。覆盖港股、美股、A 股、日韩印台、加密货币与外汇，提供 REST 与 WebSocket。

完整示例与分接口用法：[使用说明](USAGE_CN.md) · [USAGE.md](USAGE.md)。当前版本 **0.3.0**。

## 安装

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.3.0
```

## 快速开始

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

## REST 客户端

| 客户端 | 前缀 | 说明 |
|--------|------|------|
| `client.Stock` | `stock` | 港股、美股、A 股 |
| `client.Crypto` | `crypto` | 加密货币 |
| `client.Japan` | `japan` | 日股 |
| `client.India` | `india` | 印股 |
| `client.Korea` | `korea` | 韩股（`.KS`） |
| `client.Taiwan` | `taiwan` | 台股（`.TW`） |
| `client.Common` | `common` | 外汇、贵金属、期货 |
| `client.Basic` | -- | 标的、日历、档案 |
| `client.Packages` | -- | 当前 Key 套餐额度 |
| `client.Market` | -- | 温度、宽度、成交额、排行 |
| `client.Plate` | -- | 行业 / 概念板块 |
| `client.StockInfo` | -- | 估值、评级、公司 |
| `client.Financial` | -- | 财报、分红、盈利 |

### 行情方法（stock / crypto / japan / india / korea / taiwan / common）

| 方法 | HTTP | 路径 |
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

新闻是独立连接（`wss://data.infoway.io/news`），需要单独授权。

`OnReconnect` 只在再次连上时触发。`Close()` 不触发 `OnDisconnect`，并会唤醒退避。已退订的内容不会回放。

## 测试

```bash
go test ./...
INFOWAY_LIVE=1 INFOWAY_API_KEY=… go test ./... -count=1 -timeout 3m
```

## 许可

MIT
