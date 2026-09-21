# Infoway Go SDK 使用说明

面向量化、行情接入和基本面查询的 Go 1.22+ 使用指南。安装说明见 [README_CN.md](README_CN.md)，官方接口定义见 [docs.infoway.io](https://docs.infoway.io)。英文对照：[USAGE.md](USAGE.md)。

- 模块：`github.com/infoway-api/infoway-sdk-go`（`v0.3.0`）
- REST 默认地址：`https://data.infoway.io`
- 行情 WebSocket：`wss://data.infoway.io/ws`
- 新闻 WebSocket：`wss://data.infoway.io/news`

---

## 目录

1. [安装与客户端](#1-安装与客户端)
2. [标的代码约定](#2-标的代码约定)
3. [REST：多市场行情](#3-rest多市场行情)
4. [REST：基础信息](#4-rest基础信息)
5. [REST：市场概览](#5-rest市场概览)
6. [REST：板块](#6-rest板块)
7. [REST：个股资料](#7-rest个股资料)
8. [REST：财务](#8-rest财务)
9. [返回值](#9-返回值)
10. [WebSocket：行情](#10-websocket行情)
11. [WebSocket：新闻](#11-websocket新闻)
12. [错误与限额](#12-错误与限额)
13. [完整示例](#13-完整示例)
14. [附录：REST 路径](#附录rest-路径)

---

## 1. 安装与客户端

```bash
go get github.com/infoway-api/infoway-sdk-go@v0.3.0
```

不传 `APIKey` 时，REST 和 WebSocket 都会读 `INFOWAY_API_KEY`。所有 REST 方法第一个参数都是 `context.Context`。

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
        BaseURL:    "https://data.infoway.io", // 可选
        Timeout:    15 * time.Second,          // 默认 15s
        MaxRetries: 3,                         // 默认 3 次，指数退避
    })
    defer client.Close()

    data, err := client.Crypto.GetTrade(context.Background(), "BTCUSDT")
    if err != nil {
        panic(err)
    }
    fmt.Println(data)
}
```

| 选项 | 默认 | 含义 |
|------|------|------|
| `APIKey` | `INFOWAY_API_KEY` | API Key |
| `BaseURL` | `https://data.infoway.io` | REST 根地址 |
| `Timeout` | `15s` | 单次请求超时 |
| `MaxRetries` | `3` | 重试次数 |

| 入口 | 用途 |
|------|------|
| `Stock` / `Crypto` / `Japan` / `India` / `Korea` / `Taiwan` / `Common` | 成交、盘口、K 线 |
| `Basic` | 标的、日历、个股档案 |
| `Packages` | 当前 Key 套餐额度 |
| `Market` | 温度、宽度、成交额、排行 |
| `Plate` | 行业 / 概念板块 |
| `StockInfo` | 估值、评级、公司 |
| `Financial` | 财报、分红、盈利 |

| 常量 | 线上值 | 用于 |
|------|--------|------|
| `KlineMin1`…`KlineYear` | 1–12 | K 线 REST / WS |
| `SymbolStockUS` / `SymbolCrypto`… | `STOCK_US` / `CRYPTO`… | `Basic` / `Financial` |
| `MarketHK`… | `HK` `US` `CN` `JP` `KS` `TW` `IN` | 概览、板块、日历 |
| `LangEN` / `LangZHCN` | `en` / `zh-CN` | REST `lang` |
| `NewsEN` / `NewsZHHans`… | `en` / `zh-Hans`… | 新闻 WS |
| `PeriodFQ` / `PeriodFY` / `PeriodFH` | `fq` / `fy` / `fh` | 财务 |
| `BusinessCrypto` / `BusinessKorea`… | 行情 WS `business` | 行情 WS |
| `ScheduleEnergy`… | `ENERGY` `FOREX` `FUTURES` `METAL` `INDICES` | 交易时段过滤 |

---

## 2. 标的代码约定

| 市场 | 形式 | 正确 | 错误 |
|------|------|------|------|
| 美股 | `.US` | `AAPL.US` | `AAPL` |
| 港股 | `.HK`，5 位补零 | `00700.HK` | `700.HK` |
| 上海 | `.SH` | `600519.SH` | `600519.CN` |
| 深圳 | `.SZ` | `000001.SZ` | `000001.CN` |
| 日本 | `.JP` | `7203.JP` | |
| 韩国 | `.KS` | `005930.KS` | |
| 印度 | `.IN` | `RELIANCE.IN` | |
| 台湾 | `.TW` | `2330.TW` | |
| 加密 | 交易对 | `BTCUSDT` | |
| 外汇 / 贵金属 | 交易对 | `USDJPY` | |

`Basic` / `Financial` / `GetStockDetail` 要的是 **产品类型**，不是市场码 `US`：

`STOCK_US` `STOCK_CN` `STOCK_HK` `STOCK_JP` `STOCK_KS` `STOCK_IN` `STOCK_TW`  
`CRYPTO` `FOREX` `FUTURES` `ENERGY` `METAL` `INDICES`

`GetSymbols` 传 `market=US` 会得到 HTTP 400：`Required parameter 'type' is not present.`

---

## 3. REST：多市场行情

七个市场客户端方法相同，只是路径前缀不同。多个代码用逗号拼接。

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

成交字段：`s` 代码，`p` 价格，`v` 数量，`vw` 成交额，`t` 毫秒，`td` 方向（0 默认 / 1 买 / 2 卖）。  
K 线按标的包装，K 棒在 `respList` 里，`t` 是 **秒** 字符串。`timestamp` 只对分钟 / 小时线有效。

单标的最多 **500** 根；多标的请求会被服务端截成每标的 2 根。SDK 一律走 `POST /{market}/v2/batch_kline`。

---

## 4. REST：基础信息

日期一律 `YYYYMMDD`。

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

`GetTradingHours` 是 `GetTradingSchedule` 的废弃别名。时段接口 **没有市场过滤**。按品种过滤用 `ENERGY` / `FOREX` / `FUTURES` / `METAL` / `INDICES`。传入 `STOCK_US` 会在发请求前报错。

`Packages.GetInfo()` 对应 `GET /package/info`：`packageName`、`expireTime`、`apiNumPerSec`、`maxWsConNum`、`maxNum`、`maxYearHisData`、`allWsNum`。

---

## 5. REST：市场概览

```go
ctx := context.Background()
_, _ = client.Market.GetTemperature(ctx, infoway.JoinMarkets(infoway.MarketHK, infoway.MarketUS), "zh-CN")
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

排行 `key` 来自 `GetRankCategories`。`GetRankConfig` 已废弃（HTTP 404）。

---

## 6. REST：板块

```go
ctx := context.Background()
_, _ = client.Plate.GetIndustry(ctx, "HK", 200)
_, _ = client.Plate.GetConcept(ctx, "HK", 100)
_, _ = client.Plate.GetMembers(ctx, "IN20293.HK", 0, 50)
_, _ = client.Plate.GetIntro(ctx, "IN20293.HK")
_, _ = client.Plate.GetChart(ctx, "HK", 50)
```

---

## 7. REST：个股资料

可选 `lang`：`en` 或 `zh-CN`。传 `""` 表示不带该参数。

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

## 8. REST：财务

每个方法都要 `symbol` + `type`。报表类可带 `period_type`（`fq` / `fy` / `fh`）。空字符串表示省略。

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

港股示例：`client.Financial.GetDividend(ctx, "00700.HK", infoway.SymbolStockHK, "")`。

生产上部分财务过滤比较宽松：`type=US` 对 `AAPL.US` 仍会出数（后缀优先）；`period_type=xx` 返回 HTTP 200 空列表。`GetStockDetail(..., CRYPTO)` 是 HTTP 200 且 `data: null`。

---

## 9. 返回值

REST 方法把 JSON 解成 `any`（`[]any` 或 `map[string]any`）。数字是 `float64`，价格在线上仍是字符串（`"305.771"`）。成交 `t` 是毫秒；K 线 `t` 是秒字符串；`vw` 是成交额，不是 VWAP。盘口 `a`/`b` 是 `[[价格…],[数量…]]`。

---

## 10. WebSocket：行情

和 REST 客户端分开。`Business` 必须和市场一致；错频道会 ack，然后永远不推。

`PrintFrames` 默认 **false**。REST 和 WebSocket 都会回落到 `INFOWAY_API_KEY`。`Connect` 会阻塞，直到 context 取消、调用 `Close`，或握手 401。

```go
ws, err := infoway.NewWebSocket(infoway.WSOptions{
    APIKey:      os.Getenv("INFOWAY_API_KEY"),
    Business:    infoway.BusinessCrypto, // stock / japan / india / korea / taiwan / crypto / common
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

股票成交类型需要 `includeTy=true`。加密货币仍然没有 `ty`。回调拿到的是 **`data`**，不是整包。可以在 `Connect` 之前订阅，连上后会重放。

生命周期（重连 / 订阅 / 关闭）：

- 心跳 `10010` 每 30 秒一次，**每个存活会话只启一个 goroutine**。一次掉线最多安排 **一次** 重连（1 秒起、30 秒封顶）；成功连上后退避会重置。
- `OnReconnect` 只在**再次**连上时触发，首次 `Connect` 不会调用。
- `OnDisconnect` 只表示意外掉线。`Close()` **不**触发它、也**不**重连，并会唤醒退避等待，关闭马上返回。
- 客户端保存的是**目标订阅集**。已退订的内容重连后不会再发；重连窗口里的新订阅会在下一跳补发。K 线按周期退订，其它周期保留。
- HTTP 401 仍然停止重连。

| 现象 | 原因 |
|------|------|
| 首帧是纯文本 `You have permission...` | `business=stock` 欢迎语，SDK 会跳过 |
| `{"code":200,"msg":"ws connect success"}` | 欢迎帧，不是错误 |
| ack `ok` 然后没数据 | 频道错、代码不存在、或休市 |
| 心跳没有回复 | 服务端不回 `10010`；不要因为没 ack 就重连 |
| 连上不久被踢 | **每连接每分钟 60 帧**；代码要合并发送 |
| HTTP 401 | Key 无效；`*AuthError`，不再重连 |

协议：

| 方向 | 码 | 含义 |
|------|----|------|
| 出站 | 10000 / 10003 / 10006 | 订阅成交 / 盘口 / K 线 |
| 出站 | 11000 / 11001 / 11002 | 退订 |
| 出站 | 10010 | 心跳（30s；`ack=1` 会回 10011） |
| 入站 | 10001 / 10004 / 10007 | 订阅 ack |
| 入站 | 10002 / 10005 / 10008 | 推送 |
| 入站 | 10011 | 心跳 ack（可选） |
| 入站 | 11010 | 退订 ack（行情 + 新闻） |
| 入站 | 200 | 欢迎 |
| 入站 | 500–521 | 服务端错误 → `OnError`（501/502 限流；见 `WsErrorCode`） |

WebSocket 的 508 是 `APIKEY_EXPIRED`，不是 REST 的 `PRODUCT_NOT_EXISTS`。不要用 `WsErrorCode` 去解 REST 的 `ret`。

---

## 11. WebSocket：新闻

`wss://data.infoway.io/news`，单独授权。每个 Key 只能有一条新闻连接。默认语言是 `en`。

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
news.Unsubscribe()   // 11020
news.Subscribe("en") // 替换语言
_ = news.Close()
```

推送字段：`dk`（去重）、`country`、`lang`、`route`、`title`、`published`（**秒**）、`urgency`（越小越急）、`provider`、`symbols[]`、`link`、`content`、`sd`（摘要）。

| 方向 | 码 | 含义 |
|------|----|------|
| 出站 | 10020 / 11020 | 订阅 / 退订 |
| 入站 | 10021 / 10022 | Ack / 推送 |

没有新闻权限的 Key 会在握手阶段收到 HTTP 401。当前 `Lang` 会在重连时回放；`Unsubscribe()` 清空语言后重连不再订阅。`OnReconnect` / `OnDisconnect` 规则与行情通道相同。

---

## 12. 错误与限额

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
    // HTTP → RestErrorCode；WS → WsErrorCode。508–514 数字冲突。
    fmt.Fprintf(os.Stderr, "ret=%d name=%s msg=%s trace=%s\n", e.Ret, e.ErrorName, e.Msg, e.TraceID)
}
```

信封处理与其他官方 SDK 一致：先看 HTTP 401 / 429；只有 `{detail}` 的包体（限流可能是 HTTP 200）；RFC 7807；业务 `ret`/`code` **先于** HTTP ≥ 400；最后才是网关页。501/502 按限流重试。

`Error()` 会带上码和枚举名，例如 REST `[508 PRODUCT_NOT_EXISTS] All product not exists`，WebSocket `[508 APIKEY_EXPIRED] …`。不要用 `WsErrorCode` 去解 REST 的 `ret`。

伪造 Key 时：REST 返回 `*AuthError` `[401] Token invalid`；行情 / 新闻 WebSocket 握手 HTTP 401，**不会重连**。

### REST `ret`（`RestErrorCode`）

| 码 | 名称 | 说明 |
|----|------|------|
| 200 | SUCCESS | 成功 |
| 400 | BAD_REQUEST | commonApi 参数错误（HTTP 也是 400） |
| 500 | SERVER_ERROR | 未捕获异常；或旧实现把业务错误写成了 500 |
| 501 / 502 | REQUEST_EXCEED_LIMIT / REQUEST_FOR_DAY_LIMIT | 限流 → `*RateLimitError` |
| 503 | KLINE_EXCEEDS_LIMIT | K 线根数超限 |
| 505 | PRODUCTS_EXCEEDS_LIMIT | 品种数超限 |
| 506 / 507 | PARAM_ERROR / PARAM_LOST | 参数错误 / 缺失 |
| **508** | **PRODUCT_NOT_EXISTS** | **品种不存在**（不是 WS 的 key 过期） |
| 509 | TOKEN_PERMISSION_EXPIRED | token 权限过期 |
| 513 | TIME_LIMIT_ERROR | K 线 `timestamp` 超出历史深度 |
| **514** | **NO_PERMISSION** | **无该市场权限**（不是 WS 的 URL 错误） |

commonApi（`/common/basic/*`）只有 200 / 400 / 500。

对照生产（2026-09-21）：行情业务错误目前仍是 **HTTP 200 + `ret=500`**，英文模板在 `msg` 里（`All product not exists`、`Param error：klineType`、`Timestamp limit error…`、`Kline quantity exceeds the limit：500`）。SDK 会带上 **message**。

### WebSocket `code`（`WsErrorCode`）

| 码 | 名称 | 说明 |
|----|------|------|
| 501 / 502 | REQUEST_FREQUENCY_MIN_EXCEED / DAY | 单连接 60 帧/分钟等 → `*RateLimitError` |
| 505 / 516 | PRODUCTS_QUANTITY_EXCEED | 单连接 / 该 Key 全部连接 |
| 506 / 507 | PARAM_ERROR / PARAM_LOST | 参数错误或缺失 |
| **508–511** | **APIKEY_*** | **过期 / 无效 / 空 / 黑名单** |
| 512 / 513 / 514 | 连接超限 / 心跳超时 / URL 错误 | 513 后服务端会断开 |
| 515 | PARAM_NOT_JSON | 入站不是 JSON |
| 517–521 | 握手失败 | 缺 key / 无权限；519/520 在韩股、台股、新闻上含义不同 |

REST 限额约 **1200 次/分钟/Key**。HTTP 429、REST `ret` 501/502，或 body 为 `{"detail":"Rate limit exceeded"}` 时，SDK 会先退避再重试。

---

## 13. 完整示例

先拉一笔 REST 现价，再挂实时成交。

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
        fmt.Fprintln(os.Stderr, "请先设置环境变量 INFOWAY_API_KEY")
        os.Exit(1)
    }

    client := infoway.New(infoway.Options{APIKey: apiKey})
    defer client.Close()
    ctx := context.Background()

    trade, err := client.Crypto.GetTrade(ctx, "BTCUSDT")
    if err != nil {
        panic(err)
    }
    fmt.Println("REST 现价:", trade)

    company, err := client.StockInfo.GetCompany(ctx, "AAPL.US", "zh-CN")
    if err != nil {
        panic(err)
    }
    fmt.Println("公司资料:", company)

    earn, err := client.Financial.GetEarningStatus(ctx, "AAPL.US", infoway.SymbolStockUS)
    if err != nil {
        panic(err)
    }
    fmt.Println("盈利状态:", earn)

    pkg, err := client.Packages.GetInfo(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println("额度:", pkg)

    ws, err := infoway.NewWebSocket(infoway.WSOptions{
        APIKey:   apiKey,
        Business: infoway.BusinessCrypto,
    })
    if err != nil {
        panic(err)
    }
    first := make(chan struct{}, 1)
    ws.OnTrade = func(data map[string]any) {
        fmt.Println("WS 成交:", data)
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
        fmt.Fprintln(os.Stderr, "45 秒内没有成交推送")
    }
    _ = ws.Close()
}
```

```bash
export INFOWAY_API_KEY=你的密钥
go run .
```

实盘契约 / 深度探测（需显式打开）：

```bash
cd sdks/go
INFOWAY_LIVE=1 INFOWAY_API_KEY=你的密钥 go test ./... -count=1 -timeout 3m
```

---

## 附录：REST 路径

行情（`{market}` = `stock` / `crypto` / `japan` / `india` / `korea` / `taiwan` / `common`）：

- `GET /{market}/batch_trade/{codes}`
- `GET /{market}/batch_depth/{codes}`
- `POST /{market}/v2/batch_kline`

基础 / 财务 / 额度：

- `GET /common/basic/symbols`
- `GET /common/basic/symbols/info`
- `GET /common/basic/symbols/adjustment_factors`
- `GET /common/basic/markets/trading_days`
- `GET /common/basic/markets/trading_schedule`
- `GET /common/basic/markets`
- `GET /common/basic/stock/detail`
- `GET /common/basic/financial/{earning_status|income_statement|revenue|cash_flow|balance_sheet|statistics|dividend|dividend_payout|earnings}`
- `GET /package/info`

市场 / 板块 / 个股：

- `GET /common/v2/basic/market/{temperature|indexes}`
- `GET /common/v2/basic/market/{breadth|turnover|leaders|overview|rank/categories}/{market}`
- `GET /common/v2/basic/market/rank/{market}/{key}`
- `GET /common/v2/basic/plate/{industry|concept|chart}/{market}`
- `GET /common/v2/basic/plate/{members|intro}/{plateSymbol}`
- `GET /common/v2/basic/stock/{valuation|ratings|company|panorama|concepts|events|drivers}/{symbol}`
