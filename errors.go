package infoway

import (
	"fmt"
	"strings"
)

// APIError is a non-success Infoway response.
//
// Ret is the wire code (REST ret/code, WS code, or HTTP status).
// ErrorName is the matching RestErrorCode or WsErrorCode constant when known.
// 508–514 share numbers across the two channels but not meanings.
type APIError struct {
	Ret       int
	Msg       string
	TraceID   string
	ErrorName string
}

func (e *APIError) Error() string {
	if e == nil {
		return "infoway: api error"
	}
	prefix := fmt.Sprintf("[%d]", e.Ret)
	if e.ErrorName != "" {
		prefix = fmt.Sprintf("[%d %s]", e.Ret, e.ErrorName)
	}
	text := prefix + " " + e.Msg
	if e.TraceID != "" {
		text += " (trace: " + e.TraceID + ")"
	}
	return text
}

// OfRest classifies a REST body ret/code with RestErrorCode.
func OfRest(ret int, msg, traceID string) *APIError {
	name := RestErrorName(ret)
	display := msg
	if strings.TrimSpace(display) == "" {
		if label := RestErrorLabel(ret); label != "" {
			display = label
		} else {
			display = "API error"
		}
	}
	return &APIError{Ret: ret, Msg: display, TraceID: traceID, ErrorName: name}
}

// OfWS classifies a WebSocket error-frame code with WsErrorCode.
func OfWS(ret int, msg, traceID string) *APIError {
	name := WsErrorName(ret)
	display := msg
	if strings.TrimSpace(display) == "" {
		if label := WsErrorLabel(ret); label != "" {
			display = label
		} else {
			display = "API error"
		}
	}
	return &APIError{Ret: ret, Msg: display, TraceID: traceID, ErrorName: name}
}

// OfHTTPStatus is a gateway / transport status. It must not look up RestErrorCode
// (HTTP 502 is not daily-quota 502).
func OfHTTPStatus(status int, msg, traceID string) *APIError {
	if strings.TrimSpace(msg) == "" {
		msg = "HTTP error"
	}
	return &APIError{Ret: status, Msg: msg, TraceID: traceID}
}

// AuthError is HTTP / handshake 401.
type AuthError struct {
	*APIError
}

func newAuthError(msg, traceID string) *AuthError {
	if strings.TrimSpace(msg) == "" {
		msg = "Unauthorized"
	}
	return &AuthError{APIError: &APIError{Ret: 401, Msg: msg, TraceID: traceID}}
}

// RateLimitError is HTTP 429, HTTP 200 + detail rate-limit, or REST/WS 501/502.
type RateLimitError struct {
	*APIError
}

func restRateLimit(ret int, msg, traceID string) *RateLimitError {
	if strings.TrimSpace(msg) == "" {
		msg = "Rate limit exceeded"
	}
	name := RestErrorName(ret)
	if name == "" {
		name = "RATE_LIMIT"
	}
	return &RateLimitError{APIError: &APIError{Ret: ret, Msg: msg, TraceID: traceID, ErrorName: name}}
}

func wsRateLimit(ret int, msg, traceID string) *RateLimitError {
	if strings.TrimSpace(msg) == "" {
		msg = "Rate limit exceeded"
	}
	return &RateLimitError{APIError: OfWS(ret, msg, traceID)}
}

// TimeoutError is a request timeout after retries are exhausted (or a single timeout).
type TimeoutError struct {
	Msg string
}

func (e *TimeoutError) Error() string {
	if e == nil || e.Msg == "" {
		return "infoway: request timed out"
	}
	return e.Msg
}

// IOError is a network failure after retries are exhausted.
type IOError struct {
	Msg   string
	Cause error
}

func (e *IOError) Error() string {
	if e == nil {
		return "infoway: io error"
	}
	if e.Cause != nil {
		return e.Msg + ": " + e.Cause.Error()
	}
	return e.Msg
}

func (e *IOError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func restFailure(ret int, msg, traceID string) error {
	ret = ClassifyRest(ret, msg)
	if ret == 401 {
		return newAuthError(msg, traceID)
	}
	if ret == 429 || ret == int(RestRequestExceedLimit) || ret == int(RestRequestForDayLimit) {
		return restRateLimit(ret, msg, traceID)
	}
	return OfRest(ret, msg, traceID)
}

func wsFailure(code int, msg, traceID string) error {
	if code == 401 {
		return newAuthError(msg, traceID)
	}
	if code == 429 || code == int(WsErrRequestFrequencyMinExceed) || code == int(WsErrRequestFrequencyDayExceed) {
		return wsRateLimit(code, msg, traceID)
	}
	return OfWS(code, msg, traceID)
}
