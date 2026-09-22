package infoway

import (
	"context"
	"fmt"
	"strings"
)

const (
	maxSymbols              = 100
	maxKlineBars            = 500
	maxKlineBarsWhenBatched = 2
)

func symbolCount(codes string) int {
	seen := map[string]struct{}{}
	for _, part := range strings.Split(codes, ",") {
		token := strings.TrimSpace(part)
		if token != "" {
			seen[token] = struct{}{}
		}
	}
	return len(seen)
}

func checkSymbols(codes string) error {
	if symbolCount(codes) > maxSymbols {
		return OfRest(int(RestProductsExceedsLimit), fmt.Sprintf("Products quantity exceeds the limit：%d", maxSymbols), "")
	}
	return nil
}

func checkKline(codes string, count int) error {
	if err := checkSymbols(codes); err != nil {
		return err
	}
	if symbolCount(codes) > 1 && count > maxKlineBarsWhenBatched {
		return OfRest(int(RestParamError), fmt.Sprintf("Param error：klineNum exceeds %d when requesting multiple symbols", maxKlineBarsWhenBatched), "")
	}
	if count > maxKlineBars {
		return OfRest(int(RestKlineExceedsLimit), fmt.Sprintf("Kline quantity exceeds the limit：%d", maxKlineBars), "")
	}
	return nil
}

// MarketData is trade / depth / kline for one market prefix.
type MarketData struct {
	http   *httpClient
	prefix string
}

// GetTrade returns real-time trades for comma-separated codes.
func (m *MarketData) GetTrade(ctx context.Context, codes string) (any, error) {
	if err := checkSymbols(codes); err != nil {
		return nil, err
	}
	return m.http.get(ctx, "/"+m.prefix+"/batch_trade/"+codes, nil)
}

// GetDepth returns the transposed order book (a/b columns).
func (m *MarketData) GetDepth(ctx context.Context, codes string) (any, error) {
	if err := checkSymbols(codes); err != nil {
		return nil, err
	}
	return m.http.get(ctx, "/"+m.prefix+"/batch_depth/"+codes, nil)
}

// GetKline returns candles. timestamp is unix seconds for minute/hour bars; nil is latest.
func (m *MarketData) GetKline(ctx context.Context, codes string, klineType KlineType, count int, timestamp *int64) (any, error) {
	if err := checkKline(codes, count); err != nil {
		return nil, err
	}
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
