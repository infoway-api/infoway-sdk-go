package infoway

import (
	"context"
	"strconv"
)

// StockInfo is valuation, ratings, company overview and events.
type StockInfo struct {
	http *httpClient
}

func (s *StockInfo) GetValuation(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/valuation/"+symbol, query("lang", lang))
}

func (s *StockInfo) GetRatings(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/ratings/"+symbol, query("lang", lang))
}

func (s *StockInfo) GetCompany(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/company/"+symbol, query("lang", lang))
}

func (s *StockInfo) GetPanorama(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/panorama/"+symbol, query("lang", lang))
}

func (s *StockInfo) GetConcepts(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/concepts/"+symbol, query("lang", lang))
}

func (s *StockInfo) GetEvents(ctx context.Context, symbol string, limit int, lang string) (any, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.http.get(ctx, "/common/v2/basic/stock/events/"+symbol,
		query("limit", strconv.Itoa(limit), "lang", lang))
}

func (s *StockInfo) GetDrivers(ctx context.Context, symbol, lang string) (any, error) {
	return s.http.get(ctx, "/common/v2/basic/stock/drivers/"+symbol, query("lang", lang))
}
