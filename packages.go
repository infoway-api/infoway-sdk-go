package infoway

import "context"

// Packages is quota for the current API key.
type Packages struct {
	http *httpClient
}

func (p *Packages) GetInfo(ctx context.Context) (any, error) {
	return p.http.get(ctx, "/package/info", nil)
}
