package infoway

import (
	"context"
	"strconv"
)

// Plate is industry / concept sectors.
type Plate struct {
	http *httpClient
}

func (p *Plate) GetIndustry(ctx context.Context, market string, limit int) (any, error) {
	if limit <= 0 {
		limit = 200
	}
	return p.http.get(ctx, "/common/v2/basic/plate/industry/"+market, query("limit", strconv.Itoa(limit)))
}

func (p *Plate) GetConcept(ctx context.Context, market string, limit int) (any, error) {
	if limit <= 0 {
		limit = 100
	}
	return p.http.get(ctx, "/common/v2/basic/plate/concept/"+market, query("limit", strconv.Itoa(limit)))
}

func (p *Plate) GetMembers(ctx context.Context, plateSymbol string, offset, limit int) (any, error) {
	if limit <= 0 {
		limit = 50
	}
	return p.http.get(ctx, "/common/v2/basic/plate/members/"+plateSymbol,
		query("offset", strconv.Itoa(offset), "limit", strconv.Itoa(limit)))
}

func (p *Plate) GetIntro(ctx context.Context, plateSymbol string) (any, error) {
	return p.http.get(ctx, "/common/v2/basic/plate/intro/"+plateSymbol, nil)
}

func (p *Plate) GetChart(ctx context.Context, market string, limit int) (any, error) {
	if limit <= 0 {
		limit = 50
	}
	return p.http.get(ctx, "/common/v2/basic/plate/chart/"+market, query("limit", strconv.Itoa(limit)))
}
