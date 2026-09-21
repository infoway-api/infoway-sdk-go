package infoway

import (
	"context"
	"strconv"
)

// MarketClient is temperature, breadth, turnover, ranks and commentary.
type MarketClient struct {
	http *httpClient
}

// RankOptions is optional paging for GetRank.
type RankOptions struct {
	Sort   string
	Order  string
	Limit  int
	Offset int
	Lang   string
}

// GetTemperature defaults market to HK,US,CN when market is empty.
func (m *MarketClient) GetTemperature(ctx context.Context, market, lang string) (any, error) {
	if market == "" {
		market = "HK,US,CN"
	}
	return m.http.get(ctx, "/common/v2/basic/market/temperature", query("market", market, "lang", lang))
}

func (m *MarketClient) GetBreadth(ctx context.Context, market, lang string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/breadth/"+market, query("lang", lang))
}

func (m *MarketClient) GetTurnover(ctx context.Context, market, lang string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/turnover/"+market, query("lang", lang))
}

func (m *MarketClient) GetIndexes(ctx context.Context, lang string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/indexes", query("lang", lang))
}

func (m *MarketClient) GetLeaders(ctx context.Context, market string, limit int, lang string) (any, error) {
	if limit <= 0 {
		limit = 10
	}
	return m.http.get(ctx, "/common/v2/basic/market/leaders/"+market,
		query("limit", strconv.Itoa(limit), "lang", lang))
}

func (m *MarketClient) GetOverview(ctx context.Context, market, lang string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/overview/"+market, query("lang", lang))
}

func (m *MarketClient) GetRankCategories(ctx context.Context, market, lang string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/rank/categories/"+market, query("lang", lang))
}

func (m *MarketClient) GetRank(ctx context.Context, market, key string, opts *RankOptions) (any, error) {
	var q = query()
	if opts != nil {
		q = query("sort", opts.Sort, "order", opts.Order, "lang", opts.Lang)
		if opts.Limit > 0 {
			if q == nil {
				q = query()
			}
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			if q == nil {
				q = query()
			}
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	return m.http.get(ctx, "/common/v2/basic/market/rank/"+market+"/"+key, q)
}

// GetRankConfig is deprecated: the path returns HTTP 404.
func (m *MarketClient) GetRankConfig(ctx context.Context, market string) (any, error) {
	return m.http.get(ctx, "/common/v2/basic/market/rank-config/"+market, nil)
}
