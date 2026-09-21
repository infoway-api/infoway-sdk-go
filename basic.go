package infoway

import (
	"context"
	"fmt"
)

// Basic is symbols, calendar, schedule and single-name profile.
type Basic struct {
	http *httpClient
}

// GetSymbols lists instruments of one product type. symbols is optional.
func (b *Basic) GetSymbols(ctx context.Context, typ SymbolType, symbols string) (any, error) {
	return b.http.get(ctx, "/common/basic/symbols", query("type", string(typ), "symbols", symbols))
}

// GetSymbolInfo returns static fields for up to 500 codes.
func (b *Basic) GetSymbolInfo(ctx context.Context, typ SymbolType, symbols string) (any, error) {
	return b.http.get(ctx, "/common/basic/symbols/info", query("type", string(typ), "symbols", symbols))
}

// GetAdjustmentFactors returns forward factors. market is US/CN/HK/JP/IN/KS/TW. Days are YYYYMMDD.
func (b *Basic) GetAdjustmentFactors(ctx context.Context, symbol, market, beginDay, endDay string) (any, error) {
	return b.http.get(ctx, "/common/basic/symbols/adjustment_factors",
		query("symbol", symbol, "market", market, "beginDay", beginDay, "endDay", endDay))
}

// GetTradingDays returns trade_days / half_trade_days. Days are YYYYMMDD.
func (b *Basic) GetTradingDays(ctx context.Context, market, beginDay, endDay string) (any, error) {
	return b.http.get(ctx, "/common/basic/markets/trading_days",
		query("market", market, "beginDay", beginDay, "endDay", endDay))
}

// GetTradingSchedule returns the full non-equity session table.
func (b *Basic) GetTradingSchedule(ctx context.Context) (any, error) {
	return b.http.get(ctx, "/common/basic/markets/trading_schedule", nil)
}

// GetTradingScheduleByType filters ENERGY/FOREX/FUTURES/METAL/INDICES.
// Equity types such as STOCK_US return an error before the request.
func (b *Basic) GetTradingScheduleByType(ctx context.Context, typ string) (any, error) {
	resolved, ok := ParseScheduleType(typ)
	if !ok {
		return nil, fmt.Errorf("type must be one of ENERGY/FOREX/FUTURES/METAL/INDICES (got %q)", typ)
	}
	return b.http.get(ctx, "/common/basic/markets/trading_schedule", query("type", string(resolved)))
}

// GetTradingHours is a deprecated alias of GetTradingSchedule.
func (b *Basic) GetTradingHours(ctx context.Context) (any, error) {
	return b.GetTradingSchedule(ctx)
}

// GetMarkets returns the per-market session table. Takes no parameters.
func (b *Basic) GetMarkets(ctx context.Context) (any, error) {
	return b.http.get(ctx, "/common/basic/markets", nil)
}

// GetStockDetail is a single-name profile (logo, FIGI, ISIN, …).
func (b *Basic) GetStockDetail(ctx context.Context, typ SymbolType, symbol string) (any, error) {
	return b.http.get(ctx, "/common/basic/stock/detail", query("type", string(typ), "symbol", symbol))
}
