package infoway

import "context"

// Financial is statements, dividends and earnings.
type Financial struct {
	http *httpClient
}

func (f *Financial) get(ctx context.Context, name, symbol, typ, period string) (any, error) {
	return f.http.get(ctx, "/common/basic/financial/"+name,
		query("symbol", symbol, "type", typ, "period_type", period))
}

func (f *Financial) GetEarningStatus(ctx context.Context, symbol string, typ SymbolType) (any, error) {
	return f.get(ctx, "earning_status", symbol, string(typ), "")
}

func (f *Financial) GetIncomeStatement(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "income_statement", symbol, string(typ), string(period))
}

func (f *Financial) GetRevenue(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "revenue", symbol, string(typ), string(period))
}

func (f *Financial) GetCashFlow(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "cash_flow", symbol, string(typ), string(period))
}

func (f *Financial) GetBalanceSheet(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "balance_sheet", symbol, string(typ), string(period))
}

func (f *Financial) GetStatistics(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "statistics", symbol, string(typ), string(period))
}

func (f *Financial) GetDividend(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "dividend", symbol, string(typ), string(period))
}

func (f *Financial) GetDividendPayout(ctx context.Context, symbol string, typ SymbolType) (any, error) {
	return f.get(ctx, "dividend_payout", symbol, string(typ), "")
}

func (f *Financial) GetEarnings(ctx context.Context, symbol string, typ SymbolType, period PeriodType) (any, error) {
	return f.get(ctx, "earnings", symbol, string(typ), string(period))
}
