package invoance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(WithAPIKey("inv_test_key"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestEventsIngestRoundTrip(t *testing.T) {
	var gotPath, gotAuth, gotUA, gotCT, gotIdem, gotAccept string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		gotIdem = r.Header.Get("Idempotency-Key")
		gotAccept = r.Header.Get("Accept")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"event_id":"evt_1","ingested_at":"2026-01-02T03:04:05Z"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	resp, err := c.Events.Ingest(context.Background(), IngestEventParams{
		EventType:      "user.login",
		Payload:        map[string]any{"user_id": "u_42"},
		IdempotencyKey: "idem_abc",
		TraceID:        "trc_1",
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if resp.EventID != "evt_1" || resp.IngestedAt != "2026-01-02T03:04:05Z" {
		t.Errorf("unexpected response: %+v", resp)
	}

	if gotPath != "/v1/events" {
		t.Errorf("path = %q, want /v1/events", gotPath)
	}
	if gotAuth != "Bearer inv_test_key" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotUA != "invoance-go/"+SDKVersion {
		t.Errorf("user-agent = %q", gotUA)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotAccept != "application/json" {
		t.Errorf("accept = %q", gotAccept)
	}
	if gotIdem != "idem_abc" {
		t.Errorf("idempotency-key = %q", gotIdem)
	}
	if gotBody["event_type"] != "user.login" {
		t.Errorf("body.event_type = %v", gotBody["event_type"])
	}
	if gotBody["trace_id"] != "trc_1" {
		t.Errorf("body.trace_id = %v", gotBody["trace_id"])
	}
	payload, _ := gotBody["payload"].(map[string]any)
	if payload["user_id"] != "u_42" {
		t.Errorf("body.payload.user_id = %v", payload)
	}
}

func TestEventsListQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[],"page":1,"limit":1,"total":0,"has_more":false}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	limit := 1
	page := 2
	_, err := c.Events.List(context.Background(), ListEventsParams{
		Page:      &page,
		Limit:     &limit,
		EventType: "user.login",
		// DateFrom empty => must be skipped.
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Query values are sorted by url.Values.Encode.
	want := "event_type=user.login&limit=1&page=2"
	if gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}

func TestErrorPath429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":"rate_limited","message":"slow down"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Events.Ingest(context.Background(), IngestEventParams{
		EventType: "x", Payload: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsQuotaExceeded(err) {
		t.Errorf("expected quota exceeded, got %v", err)
	}
	e, ok := asError(err)
	if !ok {
		t.Fatal("not an *Error")
	}
	if e.StatusCode != 429 || e.ErrorCode != "rate_limited" || e.Message != "slow down" {
		t.Errorf("unexpected error fields: %+v", e)
	}
	if e.RetryAfterSeconds == nil || *e.RetryAfterSeconds != 30 {
		t.Errorf("retry-after = %v", e.RetryAfterSeconds)
	}
}

func TestErrorPath401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Events.Get(context.Background(), "evt_1")
	if !IsAuthentication(err) {
		t.Errorf("expected authentication error, got %v", err)
	}
}

const meBody = `{"valid":true,` +
	`"organization":{"id":"org_1","name":"Acme","issuer_name":"Acme Corp","primary_domain":"acme.com","domain_verified":true,"plan_tier":"growth"},` +
	`"tenant":{"id":"ten_1","name":"Acme"},` +
	`"api_key":{"id":"key_1","name":"ci","key_prefix":"inv_live_","key_last4":"ab12","scopes":["audit:read","audit:write"],"created_at":"2026-07-01T00:00:00Z","last_used_at":null},` +
	`"limits":{"rate_limit_per_sec":50}}`

func TestValidateHitsMeAndScopeFreeKeyIsValid(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(meBody))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// The key in this scenario carries only audit:* scopes; /v1/me needs no
	// scope, so validation must succeed (the old /v1/events probe could 403).
	res := c.Validate(context.Background())
	if !res.Valid || res.Reason != "" {
		t.Errorf("expected valid result, got %+v", res)
	}
	if gotMethod != "GET" || gotPath != "/v1/me" {
		t.Errorf("probe = %s %s, want GET /v1/me", gotMethod, gotPath)
	}
	if res.BaseURL != srv.URL {
		t.Errorf("BaseURL = %q, want %q", res.BaseURL, srv.URL)
	}
}

func TestValidateInvalidKey401(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	res := c.Validate(context.Background())
	if res.Valid {
		t.Errorf("401 must be invalid: %+v", res)
	}
	if res.Reason != "Authentication failed — check INVOANCE_API_KEY" {
		t.Errorf("reason = %q", res.Reason)
	}
	if gotPath != "/v1/me" {
		t.Errorf("probe path = %q, want /v1/me", gotPath)
	}
}

func TestValidateForbiddenStillValid(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":"ip_blocked"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	res := c.Validate(context.Background())
	if !res.Valid {
		t.Errorf("403 should be treated as valid key: %+v", res)
	}
	if gotPath != "/v1/me" {
		t.Errorf("probe path = %q, want /v1/me", gotPath)
	}
}

func TestMeReturnsDecodedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me" || r.Method != "GET" {
			t.Errorf("request = %s %s, want GET /v1/me", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(meBody))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me["valid"] != true {
		t.Errorf("me.valid = %v", me["valid"])
	}
	org, _ := me["organization"].(map[string]any)
	if org["id"] != "org_1" || org["plan_tier"] != "growth" {
		t.Errorf("me.organization = %v", org)
	}
	key, _ := me["api_key"].(map[string]any)
	scopes, _ := key["scopes"].([]any)
	if len(scopes) != 2 || scopes[0] != "audit:read" {
		t.Errorf("me.api_key.scopes = %v", scopes)
	}
	limits, _ := me["limits"].(map[string]any)
	if limits["rate_limit_per_sec"] != float64(50) {
		t.Errorf("me.limits = %v", limits)
	}
}

func TestGetBytesAcceptHeader(t *testing.T) {
	var gotAccept, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotCT = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x25, 0x50, 0x44, 0x46}) // %PDF
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data, err := c.Documents.GetOriginal(context.Background(), "evt_1")
	if err != nil {
		t.Fatalf("GetOriginal: %v", err)
	}
	if len(data) != 4 || data[0] != 0x25 {
		t.Errorf("unexpected bytes: %v", data)
	}
	if gotAccept != "application/octet-stream" {
		t.Errorf("accept = %q", gotAccept)
	}
	if gotCT != "" {
		t.Errorf("content-type should be dropped, got %q", gotCT)
	}
}

func TestClientRequiresAPIKey(t *testing.T) {
	t.Setenv(envAPIKey, "")
	_, err := New()
	if !IsValidation(err) {
		t.Errorf("expected validation error for missing key, got %v", err)
	}
}

func TestEventsVerifyRequiresOneOf(t *testing.T) {
	c, _ := New(WithAPIKey("k"))
	_, err := c.Events.Verify(context.Background(), "evt_1", VerifyEventParams{})
	if !IsValidation(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestBuildURL(t *testing.T) {
	cfg, err := resolveConfig(WithAPIKey("k"), WithBaseURL("https://x.example.com/"), WithAPIVersion("/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	tr := newTransport(cfg)
	got := tr.buildURL("/events", nil)
	if got != "https://x.example.com/v2/events" {
		t.Errorf("buildURL = %q", got)
	}
}
