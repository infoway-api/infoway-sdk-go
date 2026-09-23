# Infoway Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/infoway-api/infoway-sdk-go.svg)](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[English](README.md) | **中文**

Infoway 官方 Go SDK。覆盖 REST 行情、基础信息、市场概览、板块、个股、财务，以及行情 / 新闻 WebSocket。

| 项目 | 说明 |
| --- | --- |
| 模块 | [`github.com/infoway-api/infoway-sdk-go@v0.4.1`](https://pkg.go.dev/github.com/infoway-api/infoway-sdk-go) |
| 运行环境 | Go 1.22+ |
| REST | `https://data.infoway.io` |
| 行情 WebSocket | `wss://data.infoway.io/ws` |
| 新闻 WebSocket | `wss://data.infoway.io/news` |
| 接口频率 | [HTTP](https://docs.infoway.io/getting-started/api-limitation/http) · [WebSocket](https://docs.infoway.io/getting-started/api-limitation/websocket) |
| 错误码 | [HTTP](https://docs.infoway.io/getting-started/error-codes/http) · [WebSocket](https://docs.infoway.io/getting-started/error-codes/websocket) |
| 地址 | [行情地址](https://docs.infoway.io/getting-started/api-endpoints) |

未传入 `APIKey` 时读取环境变量 `INFOWAY_API_KEY`。REST 方法第一个参数为 `context.Context`，错误以返回值给出。

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [标的代码](#标的代码)
- [客户端](#客户端)
- [REST](#rest)
- [返回值](#返回值)
- [WebSocket](#websocket)
- [错误码](#错误码)
- [REST 路径](#rest-路径)

## 安装

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.4.1
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

    trades, err := client.Stock.GetTrade(context.Background(), "AAPL.US")
    if err != nil {
        panic(err)
    }
    fmt.Println(trades)
}
```

## 标的代码

| 市场 | 格式 | 正确 | 错误 |
| --- | --- | --- | --- |
| 美股 | `{代码}.US` | `AAPL.US` | `AAPL` |
| 港股 | 5 位 + `.HK` | `00700.HK` | `700.HK` |
| A 股上海 | `{代码}.SH` | `600519.SH` | `600519.CN` |
| A 股深圳 | `{代码}.SZ` | `000001.SZ` | `000001.CN` |
| 日股 | `{代码}.JP` | `7203.JP` | |
| 韩股 | `{代码}.KS` | `005930.KS` | |
| 印股 | `{代码}.IN` | `RELIANCE.IN` | |
| 台股 | `{代码}.TW` | `2330.TW` | |
| 加密货币 | 交易对 | `BTCUSDT` | |
| 外汇 | 货币对 | `USDJPY` | |

`Basic` / `Financial` 的 `type` 用品种类型（`STOCK_US`、`STOCK_CN`、`CRYPTO` 等），不要传市场码 `US`。

## 客户端

| 选项 | 默认 | 说明 |
| --- | --- | --- |
| `APIKey` | `INFOWAY_API_KEY` | API Key |
| `BaseURL` | `https://data.infoway.io` | REST 根地址 |
| `Timeout` | `15s` | 单次超时 |
| `MaxRetries` | `3` | 失败重试 |

入口：`Stock` / `Crypto` / `Japan` / `India` / `Korea` / `Taiwan` / `Common` / `Basic` / `Packages` / `Market` / `Plate` / `StockInfo` / `Financial`。

## REST

多个标的用英文逗号分隔。频率见 [HTTP接口限制](https://docs.infoway.io/getting-started/api-limitation/http)。

### 行情

| 方法 | 说明 |
| --- | --- |
| `GetTrade(ctx, codes)` | 最新成交 |
| `GetDepth(ctx, codes)` | 盘口 |
| `GetKline(ctx, codes, klineType, count, timestamp)` | K 线。最新数据传 `nil` |

```go
ctx := context.Background()
client.Stock.GetTrade(ctx, "AAPL.US,TSLA.US")
client.Crypto.GetDepth(ctx, "BTCUSDT")
client.Korea.GetTrade(ctx, "005930.KS")
client.Taiwan.GetTrade(ctx, "2330.TW")
client.Crypto.GetKline(ctx, "BTCUSDT", infoway.KlineMin1, 100, nil)
```

| 常量 | 值 | 周期 |
| --- | --- | --- |
| `KlineMin1` / `KlineMin5` / `KlineMin15` / `KlineMin30` | 1–4 | 分钟 |
| `KlineHour1` / `KlineHour2` / `KlineHour4` | 5–7 | 小时 |
| `KlineDay` / `KlineWeek` / `KlineMonth` / `KlineQuarter` / `KlineYear` | 8–12 | 日及以上 |

单标的最多 500 根；多标的时每个返回最近 2 根。`vw` 为成交额。

### 基础信息

日期 `YYYYMMDD`。

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

### 市场 / 板块 / 个股 / 财务

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

排行 `key` 来自 `GetRankCategories`。财务需要 `symbol` + `type`。`period_type`：`fq` 季报、`fy` 年报、`fh` 中报。

## 返回值

REST 解成 `any`（`[]any` 或 `map[string]any`）。数字为 `float64`，价格在线上仍是字符串。成交 `t` 为毫秒；K 线 `t` 为秒字符串；盘口 `a`/`b` 为 `[[价格…],[数量…]]`。

## WebSocket

### 行情

`Business` 必须与标的市场一致。`Connect` 会阻塞，请在 goroutine 中调用。单连接每分钟最多 60 帧，见 [WebSocket限制](https://docs.infoway.io/getting-started/api-limitation/websocket)。`Subscribe fail` / `516` 会走 `OnError` 并在同一条连接上重订；重连前先发 close 帧；本连接收过行情后约 90 秒没有新 tick 会自动重订（`StaleAfter` 可调）。

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

股票成交类型：`SubscribeTrade(codes, true)`。

| 行为 | 说明 |
| --- | --- |
| 心跳 | 每 30 秒发送 `10010`，服务端不回包 |
| 重连 | 断线后自动重连并补发当前订阅 |
| `OnReconnect` | 再次连上时触发 |
| `OnDisconnect` | 仅意外掉线。`Close()` 不触发 |
| HTTP 401 | 停止重连 |

| 方向 | 协议号 | 说明 |
| --- | --- | --- |
| 出 | 10000 / 10003 / 10006 | 订阅成交 / 盘口 / K 线 |
| 出 | 11000 / 11001 / 11002 | 退订 |
| 出 | 10010 | 心跳 |
| 入 | 10002 / 10005 / 10008 | 推送 |
| 入 | 11010 | 退订确认 |
| 入 | 200 | 连接成功 |

### 新闻

地址 `wss://data.infoway.io/news`，需单独授权。每个 Key 仅允许一条新闻连接。

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

订阅 `10020`，退订 `11020`，推送 `10022`。

## 错误码

REST 看 `ret`，WebSocket 看 `code`。`508`–`514` 两套含义不同。完整列表见 [HTTP错误码](https://docs.infoway.io/getting-started/error-codes/http) 与 [WebSocket错误码](https://docs.infoway.io/getting-started/error-codes/websocket)。

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

REST 限额约 1200 次/分钟/Key。无效 Key：REST 返回 `*AuthError`；WebSocket 握手 HTTP 401 且不重连。

| REST `ret` | 说明 |
| --- | --- |
| 200 | 成功 |
| 400 | 参数错误 |
| 500 | 服务端错误 |
| 501 / 502 | 频率超限 |
| 503 | K 线数量超限 |
| 505 | 标的数量超限 |
| 506 / 507 | 参数错误 / 缺失 |
| 508 | 标的不存在 |
| 509 | 权限过期 |
| 513 | 历史时间超出套餐 |
| 514 | 无权限 |

| WebSocket `code` | 说明 |
| --- | --- |
| 501 / 502 | 频率超限 |
| 505 / 516 | 订阅数量超限 |
| 506 / 507 | 参数错误 / 缺失 |
| 508–511 | API Key 过期 / 无效 / 为空 / 黑名单 |
| 512 | 连接数超限 |
| 513 | 心跳超时 |
| 515 | 非 JSON |
| 517–521 | 握手失败 |

## REST 路径

`{market}` = `stock` / `crypto` / `japan` / `india` / `korea` / `taiwan` / `common`。完整列表见 [行情地址](https://docs.infoway.io/getting-started/api-endpoints)。

| 接口 | 路径 |
| --- | --- |
| 最新成交 | `GET /{market}/batch_trade/{codes}` |
| 盘口 | `GET /{market}/batch_depth/{codes}` |
| K 线 | `POST /{market}/v2/batch_kline` |
| 品种 / 日历 / 财务 | `GET /common/basic/*` |
| 市场 / 板块 / 个股 | `GET /common/v2/basic/*` |
| 套餐 | `GET /package/info` |

## 许可证

MIT。API Key：[infoway.io](https://infoway.io)。
