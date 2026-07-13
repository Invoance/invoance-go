package invoance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuditOrgsUpdateRename(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"aorg_1","organization_id":"org_acme","name":"Acme Renamed","retention_days":365,"created_at":"2026-01-02T03:04:05Z","archived_at":null}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	name := "Acme Renamed"
	resp, err := c.Audit.Orgs.Update(context.Background(), "aorg_1", UpdateAuditOrgParams{Name: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/v1/audit/orgs/aorg_1" {
		t.Errorf("path = %q, want /v1/audit/orgs/aorg_1", gotPath)
	}
	if gotBody["name"] != "Acme Renamed" {
		t.Errorf("body.name = %v", gotBody["name"])
	}
	if resp["name"] != "Acme Renamed" || resp["id"] != "aorg_1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAuditOrgsUpdateClearName(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"aorg_1","organization_id":"org_acme","name":null,"archived_at":null}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Audit.Orgs.Update(context.Background(), "aorg_1", UpdateAuditOrgParams{})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// A nil Name must serialize as an explicit JSON null, not be omitted.
	v, present := gotBody["name"]
	if !present {
		t.Error("body.name must be present (as null) to clear the name")
	}
	if v != nil {
		t.Errorf("body.name = %v, want null", v)
	}
}

func TestAuditOrgsArchive(t *testing.T) {
	var gotMethod, gotPath string
	var gotBodyLen int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBodyLen = len(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"aorg_1","organization_id":"org_acme","archived_at":"2026-07-13T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	resp, err := c.Audit.Orgs.Archive(context.Background(), "aorg_1")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/audit/orgs/aorg_1/archive" {
		t.Errorf("path = %q, want /v1/audit/orgs/aorg_1/archive", gotPath)
	}
	if gotBodyLen != 0 {
		t.Errorf("body length = %d, want 0", gotBodyLen)
	}
	if resp["archived_at"] != "2026-07-13T00:00:00Z" {
		t.Errorf("archived_at = %v", resp["archived_at"])
	}
}

func TestAuditOrgsUnarchive(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"aorg_1","organization_id":"org_acme","archived_at":null}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	resp, err := c.Audit.Orgs.Unarchive(context.Background(), "aorg_1")
	if err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/audit/orgs/aorg_1/unarchive" {
		t.Errorf("path = %q, want /v1/audit/orgs/aorg_1/unarchive", gotPath)
	}
	if v, present := resp["archived_at"]; !present || v != nil {
		t.Errorf("archived_at = %v (present=%v), want null", v, present)
	}
}

func TestAuditOrgsDelete(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted":true,"id":"aorg_1"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	resp, err := c.Audit.Orgs.Delete(context.Background(), "aorg_1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/v1/audit/orgs/aorg_1" {
		t.Errorf("path = %q, want /v1/audit/orgs/aorg_1", gotPath)
	}
	if resp["deleted"] != true || resp["id"] != "aorg_1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAuditOrgsDeleteConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"error":"org_not_deletable"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Audit.Orgs.Delete(context.Background(), "aorg_1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsConflict(err) {
		t.Errorf("expected conflict error, got %v", err)
	}
	e, ok := asError(err)
	if !ok {
		t.Fatal("not an *Error")
	}
	if e.StatusCode != 409 || e.ErrorCode != "org_not_deletable" {
		t.Errorf("unexpected error fields: %+v", e)
	}
}

func TestAuditOrgsListIncludeArchived(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orgs":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Audit.Orgs.List(context.Background(), ListAuditOrgsParams{IncludeArchived: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotQuery != "include_archived=true" {
		t.Errorf("query = %q, want include_archived=true", gotQuery)
	}
}

func TestAuditOrgsListDefaultOmitsIncludeArchived(t *testing.T) {
	var gotQuery, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orgs":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Audit.Orgs.List(context.Background(), ListAuditOrgsParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotPath != "/v1/audit/orgs" {
		t.Errorf("path = %q, want /v1/audit/orgs", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (archived excluded by default)", gotQuery)
	}
}
