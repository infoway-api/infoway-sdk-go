package infoway

import "time"

// Options configures a Client.
type Options struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// Client is the Infoway REST entry point.
type Client struct {
	http      *httpClient
	Stock     *MarketData
	Crypto    *MarketData
	Japan     *MarketData // Tokyo: 7203.JP. Yahoo 7203.T returns 508.
	India     *MarketData // India: RELIANCE.IN. Yahoo RELIANCE.NS returns 508.
	Korea     *MarketData
	Taiwan    *MarketData
	Common    *MarketData
	Basic     *Basic
	Packages  *Packages
	Market    *MarketClient
	Plate     *Plate
	StockInfo *StockInfo
	Financial *Financial
}

// New builds a REST client. APIKey falls back to INFOWAY_API_KEY.
func New(opts Options) *Client {
	h := newHTTPClient(opts)
	return &Client{
		http:      h,
		Stock:     &MarketData{http: h, prefix: "stock"},
		Crypto:    &MarketData{http: h, prefix: "crypto"},
		Japan:     &MarketData{http: h, prefix: "japan"},
		India:     &MarketData{http: h, prefix: "india"},
		Korea:     &MarketData{http: h, prefix: "korea"},
		Taiwan:    &MarketData{http: h, prefix: "taiwan"},
		Common:    &MarketData{http: h, prefix: "common"},
		Basic:     &Basic{http: h},
		Packages:  &Packages{http: h},
		Market:    &MarketClient{http: h},
		Plate:     &Plate{http: h},
		StockInfo: &StockInfo{http: h},
		Financial: &Financial{http: h},
	}
}

// Close rejects later calls and releases idle HTTP connections.
func (c *Client) Close() {
	if c == nil || c.http == nil {
		return
	}
	c.http.closed.Store(true)
	if c.http.client != nil {
		c.http.client.CloseIdleConnections()
	}
}
