package infoway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultBaseURL = "https://data.infoway.io"
	defaultTimeout = 15 * time.Second
	defaultRetries = 3
	maxHTTPBackoff = 8 * time.Second
)

func resolveAPIKey(apiKey string) string {
	trimmed := strings.TrimSpace(apiKey)
	if apiKey != "" && trimmed == "" {
		return ""
	}
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(os.Getenv("INFOWAY_API_KEY"))
}

func requireAPIKey(apiKey string) (string, error) {
	key := resolveAPIKey(apiKey)
	if key == "" {
		return "", fmt.Errorf("apiKey is required (set INFOWAY_API_KEY or pass APIKey)")
	}
	return key, nil
}

type httpClient struct {
	apiKey     string
	baseURL    string
	maxRetries int
	client     *http.Client
	closed     atomic.Bool
}

func newHTTPClient(opts Options) *httpClient {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	retries := opts.MaxRetries
	if retries <= 0 {
		retries = defaultRetries
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	return &httpClient{
		apiKey:     resolveAPIKey(opts.APIKey),
		baseURL:    base,
		maxRetries: retries,
		client:     &http.Client{Timeout: timeout},
	}
}

func (c *httpClient) ensureOpen() error {
	if c.closed.Load() {
		return &IOError{Msg: "InfowayClient is closed"}
	}
	return nil
}

func (c *httpClient) get(ctx context.Context, path string, params url.Values) (any, error) {
	if err := c.ensureOpen(); err != nil {
		return nil, err
	}
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.execute(req)
}

func (c *httpClient) post(ctx context.Context, path string, body any) (any, error) {
	if err := c.ensureOpen(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(raw)), nil
	}
	return c.execute(req)
}

func (c *httpClient) execute(req *http.Request) (any, error) {
	req.Header.Set("apiKey", c.apiKey)
	var last error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 && req.GetBody != nil {
			body, gerr := req.GetBody()
			if gerr != nil {
				return nil, gerr
			}
			req.Body = body
		}
		resp, err := c.client.Do(req)
		if err != nil {
			if req.Context().Err() != nil {
				return nil, req.Context().Err()
			}
			if isTimeout(err) {
				last = &TimeoutError{Msg: err.Error()}
			} else {
				last = err
			}
		} else {
			data, herr := handleResponse(resp)
			_ = resp.Body.Close()
			if herr == nil {
				return data, nil
			}
			var rl *RateLimitError
			if asRateLimit(herr, &rl) {
				last = herr
			} else {
				var api *APIError
				if asAPI(herr, &api) {
					return nil, herr
				}
				var auth *AuthError
				if asAuth(herr, &auth) {
					return nil, herr
				}
				return nil, herr
			}
		}
		if attempt < c.maxRetries-1 {
			d := time.Duration(1<<attempt) * time.Second
			if d > maxHTTPBackoff {
				d = maxHTTPBackoff
			}
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(d):
			}
		}
	}
	if last != nil {
		if _, ok := last.(*RateLimitError); ok {
			return nil, last
		}
		if _, ok := last.(*TimeoutError); ok {
			return nil, last
		}
	}
	return nil, &IOError{Msg: fmt.Sprintf("Request failed after %d retries", c.maxRetries), Cause: last}
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func asRateLimit(err error, dest **RateLimitError) bool {
	if e, ok := err.(*RateLimitError); ok {
		*dest = e
		return true
	}
	return false
}

func asAPI(err error, dest **APIError) bool {
	switch e := err.(type) {
	case *APIError:
		*dest = e
		return true
	case *AuthError:
		*dest = e.APIError
		return true
	case *RateLimitError:
		*dest = e.APIError
		return true
	}
	return false
}

func asAuth(err error, dest **AuthError) bool {
	if e, ok := err.(*AuthError); ok {
		*dest = e
		return true
	}
	return false
}

func handleResponse(resp *http.Response) (any, error) {
	raw, _ := io.ReadAll(resp.Body)
	status := resp.StatusCode
	var parsed any
	var obj map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &parsed); err == nil {
			if m, ok := parsed.(map[string]any); ok {
				obj = m
			}
		}
	}
	traceID := strField(obj, "traceId")
	if traceID == "" {
		traceID = strField(obj, "trace")
	}

	if status == 401 {
		return nil, newAuthError(errorMessage(obj, raw, "Unauthorized"), traceID)
	}
	if status == 429 {
		return nil, restRateLimit(429, errorMessage(obj, raw, "Rate limit exceeded"), traceID)
	}

	if obj != nil {
		_, hasRet := obj["ret"]
		_, hasCode := obj["code"]
		_, hasData := obj["data"]
		if detail, ok := obj["detail"].(string); ok && !hasRet && !hasCode && !hasData {
			if strings.Contains(strings.ToLower(detail), "rate limit") {
				return nil, &RateLimitError{APIError: &APIError{Ret: status, Msg: detail, TraceID: traceID, ErrorName: "RATE_LIMIT"}}
			}
			return nil, OfHTTPStatus(status, detail, traceID)
		}
		if _, hasTitle := obj["title"]; hasTitle {
			if _, hasStatus := obj["status"]; hasStatus {
				problem := intField(obj, "status", status)
				detail := strField(obj, "detail")
				if detail == "" {
					detail = strField(obj, "title")
				}
				return nil, OfHTTPStatus(problem, detail, traceID)
			}
		}
		if ret, ok := businessRet(obj); ok && ret != 200 {
			return nil, restFailure(ret, errorMessage(obj, raw, "API error"), traceID)
		}
	}

	if status >= 400 {
		return nil, OfHTTPStatus(status, errorMessage(obj, raw, bodySnippet(raw, status)), traceID)
	}

	if obj == nil {
		if parsed != nil {
			return parsed, nil
		}
		return nil, OfHTTPStatus(status, "Non-JSON response: "+bodySnippet(raw, status), traceID)
	}
	if _, ok := obj["data"]; ok {
		return obj["data"], nil
	}
	return obj, nil
}

func businessRet(obj map[string]any) (int, bool) {
	if obj == nil {
		return 0, false
	}
	if v, ok := asInt(obj["ret"]); ok {
		return v, true
	}
	if v, ok := asInt(obj["code"]); ok {
		return v, true
	}
	return 0, false
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func strField(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	v, ok := obj[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func intField(obj map[string]any, key string, fallback int) int {
	if v, ok := asInt(obj[key]); ok {
		return v
	}
	return fallback
}

func errorMessage(obj map[string]any, raw []byte, fallback string) string {
	if obj != nil {
		for _, key := range []string{"msg", "message", "detail", "title"} {
			if s := strField(obj, key); s != "" {
				return s
			}
		}
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		return fallback + ": " + bodySnippet(raw, 0)
	}
	return fallback
}

func bodySnippet(raw []byte, status int) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return fmt.Sprintf("HTTP %d with empty body", status)
	}
	if len(trimmed) > 300 {
		return trimmed[:300] + "…"
	}
	return trimmed
}

func query(pairs ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			v.Set(pairs[i], pairs[i+1])
		}
	}
	if len(v) == 0 {
		return nil
	}
	return v
}
