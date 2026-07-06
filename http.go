package invoance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// transport is the low-level HTTP layer. Zero external dependencies.
type transport struct {
	cfg     config
	headers map[string]string
}

func newTransport(cfg config) *transport {
	h := map[string]string{
		"Authorization": "Bearer " + cfg.apiKey,
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"User-Agent":    "invoance-go/" + SDKVersion,
	}
	for k, v := range cfg.extraHeaders {
		h[k] = v
	}
	return &transport{cfg: cfg, headers: h}
}

// buildURL constructs {baseURL}/{apiVersion}{path}?{query}. path begins with
// "/". Nil query values are skipped.
func (t *transport) buildURL(path string, params map[string]any) string {
	base := t.cfg.baseURL + "/" + t.cfg.apiVersion + path
	if len(params) == 0 {
		return base
	}
	q := url.Values{}
	for k, v := range params {
		if v == nil {
			continue
		}
		s := stringifyParam(v)
		if s == "" {
			// pointer helpers pass nil for absent; empty string means omit.
			continue
		}
		q.Set(k, s)
	}
	enc := q.Encode()
	if enc == "" {
		return base
	}
	return base + "?" + enc
}

// stringifyParam renders a query value. Pointers are dereferenced; nil-ish
// values yield "" (skip).
func stringifyParam(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case *string:
		if x == nil {
			return ""
		}
		return *x
	case int:
		return strconv.Itoa(x)
	case *int:
		if x == nil {
			return ""
		}
		return strconv.Itoa(*x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func (t *transport) get(ctx context.Context, path string, params map[string]any, out any) error {
	return t.do(ctx, http.MethodGet, path, params, nil, "", out, false)
}

// getRaw returns the decoded JSON value untyped (into out, typically a
// *map[string]any or *any).
func (t *transport) getRaw(ctx context.Context, path string, out any) error {
	return t.do(ctx, http.MethodGet, path, nil, nil, "", out, false)
}

func (t *transport) post(ctx context.Context, path string, body map[string]any, idempotencyKey string, out any) error {
	return t.do(ctx, http.MethodPost, path, nil, body, idempotencyKey, out, false)
}

func (t *transport) put(ctx context.Context, path string, body map[string]any, out any) error {
	return t.do(ctx, http.MethodPut, path, nil, body, "", out, false)
}

func (t *transport) delete(ctx context.Context, path string, out any) error {
	return t.do(ctx, http.MethodDelete, path, nil, nil, "", out, false)
}

// getBytes performs a GET that returns raw bytes with Accept:
// application/octet-stream (Content-Type dropped).
func (t *transport) getBytes(ctx context.Context, path string) ([]byte, error) {
	reqCtx := &RequestContext{Method: http.MethodGet, Path: path}
	fullURL := t.buildURL(path, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, &Error{Kind: KindNetwork, Message: err.Error(), Request: reqCtx, cause: err}
	}
	for k, v := range t.headers {
		if k == "Content-Type" {
			continue
		}
		req.Header.Set(k, v)
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := t.cfg.httpClient.Do(req)
	if err != nil {
		return nil, t.transportError(err, reqCtx)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body := decodeJSONMap(raw)
		return nil, errorForStatus(resp.StatusCode, body, reqCtx, parseRetryAfter(resp.Header.Get("Retry-After")))
	}
	return raw, nil
}

// do executes a request and decodes a JSON response into out (which may be
// nil to discard the body).
func (t *transport) do(ctx context.Context, method, path string, params map[string]any, body map[string]any, idempotencyKey string, out any, _ bool) error {
	reqCtx := &RequestContext{Method: method, Path: path}
	fullURL := t.buildURL(path, params)

	var reqBody io.Reader
	if body != nil {
		enc, err := json.Marshal(body)
		if err != nil {
			return &Error{Kind: KindValidation, Message: err.Error(), Request: reqCtx, cause: err}
		}
		reqBody = bytes.NewReader(enc)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return &Error{Kind: KindNetwork, Message: err.Error(), Request: reqCtx, cause: err}
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if idempotencyKey == "" {
		idempotencyKey = t.cfg.idempotencyKey
	}
	if idempotencyKey != "" && (method == http.MethodPost || method == http.MethodPut) {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := t.cfg.httpClient.Do(req)
	if err != nil {
		return t.transportError(err, reqCtx)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	bodyMap := decodeJSONMap(raw)

	if apiErr := errorForStatus(resp.StatusCode, bodyMap, reqCtx, parseRetryAfter(resp.Header.Get("Retry-After"))); apiErr != nil {
		return apiErr
	}

	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &Error{
			Kind:       KindUnknown,
			Message:    "failed to decode response body: " + err.Error(),
			StatusCode: resp.StatusCode,
			Request:    reqCtx,
			cause:      err,
		}
	}
	return nil
}

// transportError classifies a transport-level failure as Timeout or Network.
func (t *transport) transportError(err error, reqCtx *RequestContext) *Error {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return &Error{
			Kind:    KindTimeout,
			Message: fmt.Sprintf("Request timed out on %s %s: %s", reqCtx.Method, reqCtx.Path, err.Error()),
			Request: reqCtx,
			cause:   err,
		}
	}
	// url.Error wrapping a timeout.
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return &Error{
			Kind:    KindTimeout,
			Message: fmt.Sprintf("Request timed out on %s %s: %s", reqCtx.Method, reqCtx.Path, err.Error()),
			Request: reqCtx,
			cause:   err,
		}
	}
	return &Error{
		Kind:    KindNetwork,
		Message: fmt.Sprintf("Network failure on %s %s: %s", reqCtx.Method, reqCtx.Path, err.Error()),
		Request: reqCtx,
		cause:   err,
	}
}

// decodeJSONMap best-effort decodes raw bytes into a JSON object, returning
// nil when the body is empty or not a JSON object.
func decodeJSONMap(raw []byte) map[string]any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// parseRetryAfter parses a Retry-After header: numeric => seconds; HTTP-date
// => delta seconds from now (floored at 0). Returns nil when absent/invalid.
func parseRetryAfter(value string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if secs, err := strconv.ParseFloat(value, 64); err == nil && secs >= 0 {
		return &secs
	}
	if ts, err := http.ParseTime(value); err == nil {
		delta := time.Until(ts).Seconds()
		if delta < 0 {
			delta = 0
		}
		return &delta
	}
	return nil
}
