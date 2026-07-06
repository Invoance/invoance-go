package invoance

import (
	"errors"
	"fmt"
)

// ErrorKind classifies an *Error. Use the Is* predicate helpers to test a
// returned error's kind through the error chain.
type ErrorKind string

const (
	// KindAuthentication maps to HTTP 401 — bad or missing API key.
	KindAuthentication ErrorKind = "authentication"
	// KindForbidden maps to HTTP 403 — authenticated but not permitted.
	KindForbidden ErrorKind = "forbidden"
	// KindNotFound maps to HTTP 404.
	KindNotFound ErrorKind = "not_found"
	// KindValidation maps to HTTP 400, and is also used for client-side
	// input validation failures raised before a request is sent.
	KindValidation ErrorKind = "validation"
	// KindConflict maps to HTTP 409.
	KindConflict ErrorKind = "conflict"
	// KindQuotaExceeded maps to HTTP 429 — rate limited / quota exceeded.
	KindQuotaExceeded ErrorKind = "quota_exceeded"
	// KindServer maps to any 5xx response.
	KindServer ErrorKind = "server"
	// KindNetwork is a transport failure before a response was received
	// (DNS, connection refused, TLS handshake, etc.).
	KindNetwork ErrorKind = "network"
	// KindTimeout means the request exceeded the configured timeout.
	KindTimeout ErrorKind = "timeout"
	// KindUnknown is any non-2xx status that does not map to a more
	// specific kind (and is below 500).
	KindUnknown ErrorKind = "unknown"
)

// RequestContext identifies the request that produced an error.
type RequestContext struct {
	Method string
	Path   string
}

// Error is the single error type returned by every SDK method. Inspect Kind
// (or use the Is* predicates) to branch on the failure category.
type Error struct {
	// Kind classifies the error.
	Kind ErrorKind
	// Message is a human-readable description.
	Message string
	// StatusCode is the HTTP status code, or 0 for client-side / transport
	// errors that never received a response.
	StatusCode int
	// ErrorCode is the server-provided machine code (body["error"]), or "".
	ErrorCode string
	// Body is the parsed JSON error body, when the server returned one.
	Body map[string]any
	// RetryAfterSeconds is populated on 429 responses that carry a
	// Retry-After header.
	RetryAfterSeconds *float64
	// Request identifies the originating request, when known.
	Request *RequestContext
	// cause is an optional underlying error (transport failures).
	cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause, if any, so errors.Is/As can traverse
// to transport errors.
func (e *Error) Unwrap() error {
	return e.cause
}

func describeRequest(ctx *RequestContext) string {
	if ctx == nil {
		return ""
	}
	return fmt.Sprintf(" on %s %s", ctx.Method, ctx.Path)
}

// kindForStatus maps an HTTP status code to an ErrorKind.
func kindForStatus(status int) ErrorKind {
	switch status {
	case 400:
		return KindValidation
	case 401:
		return KindAuthentication
	case 403:
		return KindForbidden
	case 404:
		return KindNotFound
	case 409:
		return KindConflict
	case 429:
		return KindQuotaExceeded
	}
	if status >= 500 {
		return KindServer
	}
	return KindUnknown
}

// errorForStatus returns an *Error for a non-2xx response, or nil for 2xx.
// body may be nil when the response carried no JSON.
func errorForStatus(status int, body map[string]any, req *RequestContext, retryAfter *float64) *Error {
	if status >= 200 && status < 300 {
		return nil
	}

	errorCode := "unknown"
	if body != nil {
		if v, ok := body["error"].(string); ok {
			errorCode = v
		}
	}

	var message string
	if serverMsg, ok := stringField(body, "message"); ok {
		message = serverMsg
	} else if status == 429 && retryAfter != nil {
		message = fmt.Sprintf("HTTP 429%s — rate limited, retry after %ss",
			describeRequest(req), formatSeconds(*retryAfter))
	} else {
		message = fmt.Sprintf("HTTP %d%s (no response body)", status, describeRequest(req))
	}

	return &Error{
		Kind:              kindForStatus(status),
		Message:           message,
		StatusCode:        status,
		ErrorCode:         errorCode,
		Body:              body,
		RetryAfterSeconds: retryAfter,
		Request:           req,
	}
}

func stringField(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key].(string)
	return v, ok && v != ""
}

// formatSeconds renders a retry-after value without a trailing ".0" for whole
// numbers, matching the reference SDK's string formatting.
func formatSeconds(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

// asError extracts an *Error from an error chain.
func asError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func isKind(err error, kind ErrorKind) bool {
	if e, ok := asError(err); ok {
		return e.Kind == kind
	}
	return false
}

// IsAuthentication reports whether err is an authentication (401) error.
func IsAuthentication(err error) bool { return isKind(err, KindAuthentication) }

// IsForbidden reports whether err is a forbidden (403) error.
func IsForbidden(err error) bool { return isKind(err, KindForbidden) }

// IsNotFound reports whether err is a not-found (404) error.
func IsNotFound(err error) bool { return isKind(err, KindNotFound) }

// IsValidation reports whether err is a validation error (400 or client-side).
func IsValidation(err error) bool { return isKind(err, KindValidation) }

// IsConflict reports whether err is a conflict (409) error.
func IsConflict(err error) bool { return isKind(err, KindConflict) }

// IsQuotaExceeded reports whether err is a quota/rate-limit (429) error.
func IsQuotaExceeded(err error) bool { return isKind(err, KindQuotaExceeded) }

// IsServer reports whether err is a server (5xx) error.
func IsServer(err error) bool { return isKind(err, KindServer) }

// IsNetwork reports whether err is a transport/network error.
func IsNetwork(err error) bool { return isKind(err, KindNetwork) }

// IsTimeout reports whether err is a timeout error.
func IsTimeout(err error) bool { return isKind(err, KindTimeout) }
