package infoway

import (
	"context"
	"fmt"
)

// MarketData is trade / depth / kline for one market prefix.
type MarketData struct {
	http   *httpClient
	prefix string
}

// GetTrade returns real-time trades for comma-separated codes.
func (m *MarketData) GetTrade(ctx context.Context, codes string) (any, error) {
	return m.http.get(ctx, "/"+m.prefix+"/batch_trade/"+codes, nil)
}

// GetDepth returns the transposed order book (a/b columns).
func (m *MarketData) GetDepth(ctx context.Context, codes string) (any, error) {
	return m.http.get(ctx, "/"+m.prefix+"/batch_depth/"+codes, nil)
}

// GetKline returns candles. timestamp is unix seconds for minute/hour bars; nil is latest.
func (m *MarketData) GetKline(ctx context.Context, codes string, klineType KlineType, count int, timestamp *int64) (any, error) {
	body := map[string]any{
		"codes":     codes,
		"klineType": int(klineType),
		"klineNum":  count,
	}
	if timestamp != nil {
		body["timestamp"] = *timestamp
	}
	return m.http.post(ctx, fmt.Sprintf("/%s/v2/batch_kline", m.prefix), body)
}

// Ptr is a convenience for optional timestamp / paging values.
func Ptr[T any](v T) *T { return &v }
