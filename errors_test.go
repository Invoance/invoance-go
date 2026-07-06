package invoance

import (
	"testing"
)

func TestKindForStatus(t *testing.T) {
	cases := []struct {
		status int
		kind   ErrorKind
	}{
		{400, KindValidation},
		{401, KindAuthentication},
		{403, KindForbidden},
		{404, KindNotFound},
		{409, KindConflict},
		{429, KindQuotaExceeded},
		{418, KindUnknown},
		{500, KindServer},
		{503, KindServer},
	}
	for _, c := range cases {
		if got := kindForStatus(c.status); got != c.kind {
			t.Errorf("kindForStatus(%d) = %q, want %q", c.status, got, c.kind)
		}
	}
}

func TestErrorForStatus(t *testing.T) {
	// 2xx -> nil.
	if errorForStatus(200, nil, nil, nil) != nil {
		t.Error("expected nil for 200")
	}

	// Server message wins.
	req := &RequestContext{Method: "POST", Path: "/events"}
	e := errorForStatus(400, map[string]any{"error": "bad_request", "message": "field x required"}, req, nil)
	if e == nil || e.Kind != KindValidation {
		t.Fatalf("unexpected: %+v", e)
	}
	if e.Message != "field x required" || e.ErrorCode != "bad_request" {
		t.Errorf("unexpected message/code: %q / %q", e.Message, e.ErrorCode)
	}
	if e.StatusCode != 400 {
		t.Errorf("statusCode = %d", e.StatusCode)
	}

	// 429 with retry-after and no message -> synthesized message.
	ra := 12.0
	e2 := errorForStatus(429, nil, req, &ra)
	want := "HTTP 429 on POST /events — rate limited, retry after 12s"
	if e2.Message != want {
		t.Errorf("429 message = %q, want %q", e2.Message, want)
	}
	if e2.RetryAfterSeconds == nil || *e2.RetryAfterSeconds != 12.0 {
		t.Error("retryAfter not propagated")
	}
	if e2.ErrorCode != "unknown" {
		t.Errorf("errorCode = %q, want unknown", e2.ErrorCode)
	}

	// No body, no retry-after.
	e3 := errorForStatus(500, nil, req, nil)
	want3 := "HTTP 500 on POST /events (no response body)"
	if e3.Message != want3 {
		t.Errorf("500 message = %q, want %q", e3.Message, want3)
	}
}

func TestPredicates(t *testing.T) {
	e := &Error{Kind: KindQuotaExceeded}
	if !IsQuotaExceeded(e) {
		t.Error("IsQuotaExceeded false")
	}
	if IsAuthentication(e) {
		t.Error("IsAuthentication true")
	}
	// nil / non-SDK errors.
	if IsQuotaExceeded(nil) {
		t.Error("IsQuotaExceeded(nil) true")
	}
}

func TestFormatSeconds(t *testing.T) {
	if s := formatSeconds(12.0); s != "12" {
		t.Errorf("formatSeconds(12.0) = %q", s)
	}
	if s := formatSeconds(1.5); s != "1.5" {
		t.Errorf("formatSeconds(1.5) = %q", s)
	}
}
