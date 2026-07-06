package invoance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// AuditResource is the audit-logs resource (client.Audit), with five
// sub-resources.
type AuditResource struct {
	// Events is the audit-event ledger sub-resource.
	Events *AuditEventsResource
	// Orgs is the end-customer orgs sub-resource.
	Orgs *AuditOrgsResource
	// Streams is the SIEM/webhook streams sub-resource.
	Streams *AuditStreamsResource
	// PortalSessions is the hosted-viewer portal-link sub-resource.
	PortalSessions *AuditPortalSessionsResource
	// Exports is the async export-jobs sub-resource.
	Exports *AuditExportsResource
}

func newAuditResource(t *transport) *AuditResource {
	return &AuditResource{
		Events:         &AuditEventsResource{t: t},
		Orgs:           &AuditOrgsResource{t: t},
		Streams:        &AuditStreamsResource{t: t},
		PortalSessions: &AuditPortalSessionsResource{t: t},
		Exports:        &AuditExportsResource{t: t},
	}
}

// ContentIdempotencyKey derives a stable Idempotency-Key from an event body:
// "idem_" + sha256hex(stableStringify(body)), where stableStringify is
// compact JSON with all object keys sorted deeply (NO null stripping). Because
// encoding/json already marshals map keys in sorted order and produces compact
// output, marshaling a map[string]any body yields the stable form.
func ContentIdempotencyKey(body map[string]any) string {
	stable := stableStringify(body)
	sum := sha256.Sum256(stable)
	return "idem_" + hex.EncodeToString(sum[:])
}

// stableStringify marshals a value to compact JSON with all map keys sorted
// deeply. encoding/json sorts map[string]any keys and emits compact bytes by
// default, so a plain Marshal is the stable form.
func stableStringify(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

// AuditEventsResource is the audit-event ledger sub-resource
// (client.Audit.Events).
type AuditEventsResource struct {
	t *transport
}

// IngestAuditEventParams are the parameters for Audit.Events.Ingest.
type IngestAuditEventParams struct {
	// OrganizationID is your external org id (required).
	OrganizationID string
	// Action is the required action string.
	Action string
	// Actor is the required actor object.
	Actor map[string]any
	// OccurredAt is an optional RFC3339 UTC timestamp; defaults to now.
	OccurredAt string
	// Targets is the optional list of target objects; defaults to [].
	Targets []map[string]any
	// Context is optional additional context.
	Context map[string]any
	// Metadata is optional metadata.
	Metadata map[string]any
	// IdempotencyKey overrides the derived content idempotency key.
	IdempotencyKey string
}

// Ingest appends one signed audit event (POST /audit/events). The ledger
// requires an Idempotency-Key; if none is supplied, a content-stable one is
// derived from the request body.
func (r *AuditEventsResource) Ingest(ctx context.Context, params IngestAuditEventParams) (map[string]any, error) {
	occurredAt := params.OccurredAt
	if occurredAt == "" {
		occurredAt = time.Now().UTC().Format(time.RFC3339)
	}

	targets := make([]any, 0, len(params.Targets))
	for _, tgt := range params.Targets {
		targets = append(targets, tgt)
	}

	body := map[string]any{
		"organization_id": params.OrganizationID,
		"action":          params.Action,
		"occurred_at":     occurredAt,
		"actor":           params.Actor,
		"targets":         targets,
	}
	if params.Context != nil {
		body["context"] = params.Context
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	idem := params.IdempotencyKey
	if idem == "" {
		idem = ContentIdempotencyKey(body)
	}

	var out map[string]any
	err := r.t.post(ctx, "/audit/events", body, idem, &out)
	return out, err
}

// ListAuditEventsParams are the query parameters for Audit.Events.List.
type ListAuditEventsParams struct {
	OrganizationID string
	Actions        string
	ActorID        string
	TargetID       string
	// RangeStart and RangeEnd are inclusive RFC3339 bounds on occurred_at.
	RangeStart string
	RangeEnd   string
	Limit      *int
	Cursor     string
}

// List returns a keyset-paginated audit listing (GET /audit/events).
func (r *AuditEventsResource) List(ctx context.Context, params ListAuditEventsParams) (ListAuditEventsResponse, error) {
	var out ListAuditEventsResponse
	err := r.t.get(ctx, "/audit/events", map[string]any{
		"organization_id": params.OrganizationID,
		"actions":         params.Actions,
		"actor_id":        params.ActorID,
		"target_id":       params.TargetID,
		"range_start":     params.RangeStart,
		"range_end":       params.RangeEnd,
		"limit":           params.Limit,
		"cursor":          params.Cursor,
	}, &out)
	return out, err
}

// Get retrieves a single audit event (GET /audit/events/{id}).
func (r *AuditEventsResource) Get(ctx context.Context, eventID string) (AuditEvent, error) {
	var out AuditEvent
	err := r.t.get(ctx, "/audit/events/"+eventID, nil, &out)
	return out, err
}

// Verify runs the server-side verify (pinned key) for one event
// (GET /audit/events/{id}/verify), returning the raw server response.
func (r *AuditEventsResource) Verify(ctx context.Context, eventID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.get(ctx, "/audit/events/"+eventID+"/verify", nil, &out)
	return out, err
}

// AuditOrgsResource is the audit end-customer orgs sub-resource
// (client.Audit.Orgs).
type AuditOrgsResource struct {
	t *transport
}

// CreateAuditOrgParams are the parameters for Audit.Orgs.Create.
type CreateAuditOrgParams struct {
	// OrganizationID is your external org id (required).
	OrganizationID string
	// Name is an optional display name.
	Name string
}

// Create registers an end-customer org (POST /audit/orgs).
func (r *AuditOrgsResource) Create(ctx context.Context, params CreateAuditOrgParams) (map[string]any, error) {
	body := map[string]any{"organization_id": params.OrganizationID}
	if params.Name != "" {
		body["name"] = params.Name
	}
	var out map[string]any
	err := r.t.post(ctx, "/audit/orgs", body, "", &out)
	return out, err
}

// List lists orgs (GET /audit/orgs).
func (r *AuditOrgsResource) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := r.t.get(ctx, "/audit/orgs", nil, &out)
	return out, err
}

// Integrity returns the integrity report for an org
// (GET /audit/orgs/{orgID}/integrity).
func (r *AuditOrgsResource) Integrity(ctx context.Context, organizationID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.get(ctx, "/audit/orgs/"+organizationID+"/integrity", nil, &out)
	return out, err
}

// SetRetention updates the retention window for an org
// (PUT /audit/orgs/{orgID}/retention).
func (r *AuditOrgsResource) SetRetention(ctx context.Context, organizationID string, days int) (map[string]any, error) {
	var out map[string]any
	err := r.t.put(ctx, "/audit/orgs/"+organizationID+"/retention", map[string]any{"days": days}, &out)
	return out, err
}

// AuditStreamsResource is the audit streams sub-resource
// (client.Audit.Streams).
type AuditStreamsResource struct {
	t *transport
}

// CreateAuditStreamParams are the parameters for Audit.Streams.Create.
type CreateAuditStreamParams struct {
	// URL is the required delivery endpoint.
	URL string
	// Type defaults to "webhook" (v1 supports webhook only).
	Type string
}

// Create creates a stream (POST /audit/orgs/{orgID}/streams). The signing
// secret is returned once.
func (r *AuditStreamsResource) Create(ctx context.Context, organizationID string, params CreateAuditStreamParams) (map[string]any, error) {
	streamType := params.Type
	if streamType == "" {
		streamType = "webhook"
	}
	var out map[string]any
	err := r.t.post(ctx, "/audit/orgs/"+organizationID+"/streams", map[string]any{
		"type": streamType,
		"url":  params.URL,
	}, "", &out)
	return out, err
}

// List lists streams for an org (GET /audit/orgs/{orgID}/streams).
func (r *AuditStreamsResource) List(ctx context.Context, organizationID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.get(ctx, "/audit/orgs/"+organizationID+"/streams", nil, &out)
	return out, err
}

// Delete deletes a stream (DELETE /audit/orgs/{orgID}/streams/{streamID}).
func (r *AuditStreamsResource) Delete(ctx context.Context, organizationID, streamID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.delete(ctx, "/audit/orgs/"+organizationID+"/streams/"+streamID, &out)
	return out, err
}

// Test sends a test delivery to a stream
// (POST /audit/orgs/{orgID}/streams/{streamID}/test).
func (r *AuditStreamsResource) Test(ctx context.Context, organizationID, streamID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.post(ctx, "/audit/orgs/"+organizationID+"/streams/"+streamID+"/test", nil, "", &out)
	return out, err
}

// AuditPortalSessionsResource is the hosted-viewer portal sub-resource
// (client.Audit.PortalSessions).
type AuditPortalSessionsResource struct {
	t *transport
}

// CreatePortalSessionParams are the parameters for Audit.PortalSessions.Create.
type CreatePortalSessionParams struct {
	// OrganizationID is the required org id.
	OrganizationID string
	// Intent is required: "audit_logs" or "log_streams".
	Intent string
	// SessionDurationSeconds is the viewer session length (optional).
	SessionDurationSeconds *int
	// LinkDurationSeconds is the one-time link open window (optional).
	LinkDurationSeconds *int
}

// Create mints a one-time hosted-viewer link (POST /audit/portal_sessions).
func (r *AuditPortalSessionsResource) Create(ctx context.Context, params CreatePortalSessionParams) (map[string]any, error) {
	body := map[string]any{
		"organization_id": params.OrganizationID,
		"intent":          params.Intent,
	}
	if params.SessionDurationSeconds != nil {
		body["session_duration_seconds"] = *params.SessionDurationSeconds
	}
	if params.LinkDurationSeconds != nil {
		body["link_duration_seconds"] = *params.LinkDurationSeconds
	}
	var out map[string]any
	err := r.t.post(ctx, "/audit/portal_sessions", body, "", &out)
	return out, err
}

// AuditExportsResource is the async export-jobs sub-resource
// (client.Audit.Exports).
type AuditExportsResource struct {
	t *transport
}

// CreateAuditExportParams are the parameters for Audit.Exports.Create.
type CreateAuditExportParams struct {
	// OrganizationID is the required org id.
	OrganizationID string
	// Format is required: "csv" or "ndjson".
	Format string
	// Filters is an optional filter object.
	Filters map[string]any
}

// Create queues an async export job (POST /audit/exports).
func (r *AuditExportsResource) Create(ctx context.Context, params CreateAuditExportParams) (map[string]any, error) {
	body := map[string]any{
		"organization_id": params.OrganizationID,
		"format":          params.Format,
	}
	if params.Filters != nil {
		body["filters"] = params.Filters
	}
	var out map[string]any
	err := r.t.post(ctx, "/audit/exports", body, "", &out)
	return out, err
}

// Get polls an export job (GET /audit/exports/{id}). When status is "ready"
// the response carries a download_url.
func (r *AuditExportsResource) Get(ctx context.Context, exportID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.get(ctx, "/audit/exports/"+exportID, nil, &out)
	return out, err
}
