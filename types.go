package infoway

import "strings"

// KlineType is a candlestick interval (1–12).
type KlineType int

const (
	KlineMin1    KlineType = 1
	KlineMin5    KlineType = 2
	KlineMin15   KlineType = 3
	KlineMin30   KlineType = 4
	KlineHour1   KlineType = 5
	KlineHour2   KlineType = 6
	KlineHour4   KlineType = 7
	KlineDay     KlineType = 8
	KlineWeek    KlineType = 9
	KlineMonth   KlineType = 10
	KlineQuarter KlineType = 11
	KlineYear    KlineType = 12
)

// SymbolType is the product type for /common/basic/symbols and financials.
type SymbolType string

const (
	SymbolStockUS SymbolType = "STOCK_US"
	SymbolStockCN SymbolType = "STOCK_CN"
	SymbolStockHK SymbolType = "STOCK_HK"
	SymbolStockJP SymbolType = "STOCK_JP"
	SymbolStockKS SymbolType = "STOCK_KS"
	SymbolStockIN SymbolType = "STOCK_IN"
	SymbolStockTW SymbolType = "STOCK_TW"
	SymbolCrypto  SymbolType = "CRYPTO"
	SymbolForex   SymbolType = "FOREX"
	SymbolFutures SymbolType = "FUTURES"
	SymbolEnergy  SymbolType = "ENERGY"
	SymbolMetal   SymbolType = "METAL"
	SymbolIndices SymbolType = "INDICES"
)

func (s SymbolType) String() string { return string(s) }

// ValidSymbolType reports whether s is a type the financial and symbol APIs accept.
func ValidSymbolType(s SymbolType) bool {
	switch s {
	case SymbolStockUS, SymbolStockCN, SymbolStockHK, SymbolStockJP, SymbolStockKS, SymbolStockIN, SymbolStockTW,
		SymbolCrypto, SymbolForex, SymbolFutures, SymbolEnergy, SymbolMetal, SymbolIndices:
		return true
	default:
		return false
	}
}

// Market is an equity market code.
type Market string

const (
	MarketHK Market = "HK"
	MarketUS Market = "US"
	MarketCN Market = "CN"
	MarketJP Market = "JP"
	MarketKS Market = "KS"
	MarketTW Market = "TW"
	MarketIN Market = "IN"
)

func (m Market) String() string { return string(m) }

// JoinMarkets joins market codes with commas.
func JoinMarkets(markets ...Market) string {
	parts := make([]string, 0, len(markets))
	for _, m := range markets {
		if m != "" {
			parts = append(parts, string(m))
		}
	}
	return strings.Join(parts, ",")
}

// Lang is REST language for name-like fields.
type Lang string

const (
	LangEN   Lang = "en"
	LangZHCN Lang = "zh-CN"
)

func (l Lang) String() string { return string(l) }

// NewsLang is a news-channel language.
type NewsLang string

const (
	NewsEN     NewsLang = "en"
	NewsZHHans NewsLang = "zh-Hans"
	NewsZHHant NewsLang = "zh-Hant"
	NewsJA     NewsLang = "ja"
	NewsKO     NewsLang = "ko"
	NewsDE     NewsLang = "de"
	NewsFR     NewsLang = "fr"
	NewsES     NewsLang = "es"
	NewsPT     NewsLang = "pt"
	NewsRU     NewsLang = "ru"
	NewsTR     NewsLang = "tr"
)

func (l NewsLang) String() string { return string(l) }

// PeriodType is a financial-statement period.
type PeriodType string

const (
	PeriodFQ PeriodType = "fq"
	PeriodFY PeriodType = "fy"
	PeriodFH PeriodType = "fh"
)

func (p PeriodType) String() string { return string(p) }

// ScheduleType is accepted by /common/basic/markets/trading_schedule.
type ScheduleType string

const (
	ScheduleEnergy  ScheduleType = "ENERGY"
	ScheduleForex   ScheduleType = "FOREX"
	ScheduleFutures ScheduleType = "FUTURES"
	ScheduleMetal   ScheduleType = "METAL"
	ScheduleIndices ScheduleType = "INDICES"
)

func (s ScheduleType) String() string { return string(s) }

// ParseScheduleType maps ENERGY/FOREX/FUTURES/METAL/INDICES. Equity types return false.
func ParseScheduleType(value string) (ScheduleType, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ENERGY":
		return ScheduleEnergy, true
	case "FOREX":
		return ScheduleForex, true
	case "FUTURES":
		return ScheduleFutures, true
	case "METAL":
		return ScheduleMetal, true
	case "INDICES":
		return ScheduleIndices, true
	default:
		return "", false
	}
}

// RankSort is a rank-board sort key.
type RankSort string

const (
	RankSortChg RankSort = "chg"
)

func (s RankSort) String() string { return string(s) }

// SortOrder is asc/desc.
type SortOrder string

const (
	SortDesc SortOrder = "desc"
	SortAsc  SortOrder = "asc"
)

func (s SortOrder) String() string { return string(s) }

// Business is a quotes WebSocket channel.
type Business string

const (
	BusinessStock  Business = "stock"
	BusinessJapan  Business = "japan"
	BusinessIndia  Business = "india"
	BusinessKorea  Business = "korea"
	BusinessTaiwan Business = "taiwan"
	BusinessCrypto Business = "crypto"
	BusinessCommon Business = "common"
)

func (b Business) String() string { return string(b) }

// WsCode is a subscribe / push / heartbeat protocol number.
type WsCode int

const (
	WsSubTrade    WsCode = 10000
	WsSubTradeAck WsCode = 10001
	WsPushTrade   WsCode = 10002
	WsSubDepth    WsCode = 10003
	WsSubDepthAck WsCode = 10004
	WsPushDepth   WsCode = 10005
	WsSubKline    WsCode = 10006
	WsSubKlineAck WsCode = 10007
	WsPushKline   WsCode = 10008
	WsHeartbeat   WsCode = 10010
	WsHeartApply  WsCode = 10011
	WsSubNews     WsCode = 10020
	WsSubNewsAck  WsCode = 10021
	WsPushNews    WsCode = 10022
	WsUnsubTrade  WsCode = 11000
	WsUnsubDepth  WsCode = 11001
	WsUnsubKline  WsCode = 11002
	WsUnsubAck    WsCode = 11010
	WsUnsubNews   WsCode = 11020
	WsConnectOK   WsCode = 200
)

func knownWsCode(code int) bool {
	switch WsCode(code) {
	case WsSubTrade, WsSubTradeAck, WsPushTrade,
		WsSubDepth, WsSubDepthAck, WsPushDepth,
		WsSubKline, WsSubKlineAck, WsPushKline,
		WsHeartbeat, WsHeartApply,
		WsSubNews, WsSubNewsAck, WsPushNews,
		WsUnsubTrade, WsUnsubDepth, WsUnsubKline, WsUnsubAck, WsUnsubNews,
		WsConnectOK:
		return true
	default:
		return false
	}
}

// RestErrorCode is a REST business ret. 508–514 collide with WsErrorCode but mean different things.
type RestErrorCode int

const (
	RestSuccess                    RestErrorCode = 200
	RestBadRequest                 RestErrorCode = 400
	RestServerError                RestErrorCode = 500
	RestRequestExceedLimit         RestErrorCode = 501
	RestRequestForDayLimit         RestErrorCode = 502
	RestKlineExceedsLimit          RestErrorCode = 503
	RestOrderBookDepthExceedsLimit RestErrorCode = 504
	RestProductsExceedsLimit       RestErrorCode = 505
	RestParamError                 RestErrorCode = 506
	RestParamLost                  RestErrorCode = 507
	RestProductNotExists           RestErrorCode = 508
	RestTokenPermissionExpired     RestErrorCode = 509
	RestWebsocketExceedsLimit      RestErrorCode = 510
	RestWebsocketHeartbeatTimeout  RestErrorCode = 511
	RestWebsocketDisconnected      RestErrorCode = 512
	RestTimeLimitError             RestErrorCode = 513
	RestNoPermission               RestErrorCode = 514
)

var restErrorNames = map[int]string{
	200: "SUCCESS",
	400: "BAD_REQUEST",
	500: "SERVER_ERROR",
	501: "REQUEST_EXCEED_LIMIT",
	502: "REQUEST_FOR_DAY_LIMIT",
	503: "KLINE_EXCEEDS_LIMIT",
	504: "ORDER_BOOK_DEPTH_EXCEEDS_LIMIT",
	505: "PRODUCTS_EXCEEDS_LIMIT",
	506: "PARAM_ERROR",
	507: "PARAM_LOST",
	508: "PRODUCT_NOT_EXISTS",
	509: "TOKEN_PERMISSION_EXPIRED",
	510: "WEBSOCKET_EXCEEDS_LIMIT",
	511: "WEBSOCKET_HEARTBEAT_TIMEOUT",
	512: "WEBSOCKET_DISCONNECTED",
	513: "TIME_LIMIT_ERROR",
	514: "NO_PERMISSION",
}

var restErrorLabels = map[int]string{
	200: "success",
	400: "Bad request",
	500: "Server error",
	501: "Request frequency exceed the limit",
	502: "Request frequency for the day upper limit",
	503: "Kline quantity exceeds the limit",
	504: "Orderbook depth exceeds the limit",
	505: "Products quantity exceeds the limit",
	506: "Param error",
	507: "Param lost",
	508: "All product not exists",
	509: "The token permission has expired",
	510: "WebSocket connections exceeds the limit",
	511: "Websocket heartbeat timeout",
	512: "WebSocket disconnected",
	513: "Timestamp limit error",
	514: "No permission",
}

// RestErrorName returns the REST enum constant name, or "".
func RestErrorName(code int) string { return restErrorNames[code] }

// RestErrorLabel returns the REST label, or "".
func RestErrorLabel(code int) string { return restErrorLabels[code] }

// WsErrorCode is a WebSocket rejection code. Do not use it for REST ret.
type WsErrorCode int

const (
	WsErrServerError                   WsErrorCode = 500
	WsErrRequestFrequencyMinExceed     WsErrorCode = 501
	WsErrRequestFrequencyDayExceed     WsErrorCode = 502
	WsErrKlineQuantityExceed           WsErrorCode = 503
	WsErrOrderBookDepthExceed          WsErrorCode = 504
	WsErrProductsQuantityExceed        WsErrorCode = 505
	WsErrParamError                    WsErrorCode = 506
	WsErrParamLost                     WsErrorCode = 507
	WsErrAPIKeyExpired                 WsErrorCode = 508
	WsErrAPIKeyInvalid                 WsErrorCode = 509
	WsErrAPIKeyEmpty                   WsErrorCode = 510
	WsErrAPIKeyBlacklist               WsErrorCode = 511
	WsErrWSConnExceed                  WsErrorCode = 512
	WsErrWSHeartTimeout                WsErrorCode = 513
	WsErrWSURLWrong                    WsErrorCode = 514
	WsErrParamNotJSON                  WsErrorCode = 515
	WsErrAllProductsQuantityExceed     WsErrorCode = 516
	WsErrHandshakeAPIKeyMissing        WsErrorCode = 517
	WsErrHandshakeAPIKeyNotExist       WsErrorCode = 518
	WsErrHandshakeNoPermission         WsErrorCode = 519
	WsErrProductCodeOrAlreadyConnected WsErrorCode = 520
	WsErrHandshakeMaxConnections       WsErrorCode = 521
)

var wsErrorNames = map[int]string{
	500: "SERVER_ERROR",
	501: "REQUEST_FREQUENCY_MIN_EXCEED",
	502: "REQUEST_FREQUENCY_DAY_EXCEED",
	503: "KLINE_QUANTITY_EXCEED",
	504: "ORDER_BOOK_DEPTH_EXCEED",
	505: "PRODUCTS_QUANTITY_EXCEED",
	506: "PARAM_ERROR",
	507: "PARAM_LOST",
	508: "APIKEY_EXPIRED",
	509: "APIKEY_INVALID",
	510: "APIKEY_EMPTY",
	511: "APIKEY_BLACKLIST",
	512: "WS_CONN_EXCEED",
	513: "WS_HEART_TIMEOUT",
	514: "WS_URL_WRONG",
	515: "PARAM_NOT_JSON",
	516: "ALL_PRODUCTS_QUANTITY_EXCEED",
	517: "WS_HANDSHAKE_APIKEY_MISSING",
	518: "WS_HANDSHAKE_APIKEY_NOT_EXIST",
	519: "WS_HANDSHAKE_NO_PERMISSION",
	520: "PRODUCT_CODE_OR_ALREADY_CONNECTED",
	521: "WS_HANDSHAKE_MAX_CONNECTIONS",
}

var wsErrorLabels = map[int]string{
	500: "Server error",
	501: "Request frequency exceed the limit",
	502: "Request frequency for the day upper limit",
	503: "Kline quantity exceeds the limit",
	504: "Orderbook depth exceeds the limit",
	505: "Products quantity exceeds the limit",
	506: "Param error",
	507: "Param lost",
	508: "API key expired",
	509: "API key invalid",
	510: "API key empty",
	511: "API key blacklisted",
	512: "WebSocket connections exceeds the limit",
	513: "Websocket heartbeat timeout",
	514: "WebSocket URL wrong",
	515: "Param not json",
	516: "All products quantity exceeds the limit",
	517: "Handshake API key missing",
	518: "Handshake API key not exist",
	519: "Handshake no permission",
	520: "Product code mismatch or already connected",
	521: "Handshake max connections",
}

// WsErrorName returns the WS enum constant name, or "".
func WsErrorName(code int) string { return wsErrorNames[code] }

// WsErrorLabel returns the WS label, or "".
func WsErrorLabel(code int) string { return wsErrorLabels[code] }

// ClassifyRest maps a wrapped ret=500 back to the business code carried in msg.
func ClassifyRest(ret int, msg string) int {
	if ret != int(RestServerError) || strings.TrimSpace(msg) == "" {
		return ret
	}
	text := strings.ToLower(strings.TrimSpace(msg))
	for code, label := range restErrorLabels {
		if code == 200 || code == 400 || code == 500 {
			continue
		}
		if strings.HasPrefix(text, strings.ToLower(label)) {
			return code
		}
	}
	return ret
}

// IsTerminalWs reports codes a reconnect cannot fix.
func IsTerminalWs(code int) bool {
	switch WsErrorCode(code) {
	case WsErrRequestFrequencyDayExceed,
		WsErrAPIKeyExpired,
		WsErrAPIKeyInvalid,
		WsErrAPIKeyEmpty,
		WsErrAPIKeyBlacklist,
		WsErrWSConnExceed,
		WsErrWSURLWrong,
		WsErrAllProductsQuantityExceed,
		WsErrHandshakeAPIKeyMissing,
		WsErrHandshakeAPIKeyNotExist,
		WsErrHandshakeNoPermission,
		WsErrProductCodeOrAlreadyConnected,
		WsErrHandshakeMaxConnections:
		return true
	default:
		return false
	}
}

// IsWsError reports whether code is a gateway rejection (not a protocol frame).
func IsWsError(code int) bool {
	return code >= 500 && code < 10000 && !knownWsCode(code)
}

func wire(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case SymbolType:
		return string(t)
	case Market:
		return string(t)
	case Lang:
		return string(t)
	case NewsLang:
		return string(t)
	case PeriodType:
		return string(t)
	case ScheduleType:
		return string(t)
	case RankSort:
		return string(t)
	case SortOrder:
		return string(t)
	case Business:
		return string(t)
	default:
		return ""
	}
}
