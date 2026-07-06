package invoance

// This file defines the response models. JSON tags match the wire keys
// exactly. Optional fields use pointers or omitempty. Flexible JSON blobs
// (payload, metadata, context, actor, targets) use map[string]any or
// []map[string]any.

// OrganizationPublic is the tenant-org descriptor embedded in some responses.
type OrganizationPublic struct {
	Name             string  `json:"name"`
	IssuerName       string  `json:"issuer_name"`
	PrimaryDomain    string  `json:"primary_domain"`
	DomainVerified   bool    `json:"domain_verified"`
	DomainVerifiedAt *string `json:"domain_verified_at,omitempty"`
	LogoURL          *string `json:"logo_url,omitempty"`
}

// ── Events ──────────────────────────────────────────────────

// IngestEventResponse is returned by Events.Ingest.
type IngestEventResponse struct {
	EventID    string `json:"event_id"`
	IngestedAt string `json:"ingested_at"`
}

// EventListItem is one entry in a ListEventsResponse.
type EventListItem struct {
	EventID         string  `json:"event_id"`
	EventType       string  `json:"event_type"`
	PayloadHash     string  `json:"payload_hash"`
	EventHash       string  `json:"event_hash"`
	RetentionPolicy string  `json:"retention_policy"`
	IngestedAt      string  `json:"ingested_at"`
	EventTime       *string `json:"event_time,omitempty"`
	IdempotencyKey  *string `json:"idempotency_key,omitempty"`
}

// ListEventsResponse is a paginated event listing.
type ListEventsResponse struct {
	Events  []EventListItem `json:"events"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
	Total   int             `json:"total"`
	HasMore bool            `json:"has_more"`
}

// ComplianceEvent is a single compliance event.
type ComplianceEvent struct {
	EventID         string              `json:"event_id"`
	TenantID        string              `json:"tenant_id"`
	EventType       string              `json:"event_type"`
	Payload         map[string]any      `json:"payload"`
	EventTime       *string             `json:"event_time,omitempty"`
	RetentionPolicy string              `json:"retention_policy"`
	// AccessTier is not returned by every endpoint (e.g. the single-event GET omits it).
	AccessTier      *string             `json:"access_tier,omitempty"`
	ExpiresAt       *string             `json:"expires_at,omitempty"`
	APIKeyID        *string             `json:"api_key_id,omitempty"`
	UserID          *string             `json:"user_id,omitempty"`
	IngestedAt      string              `json:"ingested_at"`
	PayloadHash     string              `json:"payload_hash"`
	RequestHash     string              `json:"request_hash"`
	EventHash       string              `json:"event_hash"`
	IdempotencyKey  *string             `json:"idempotency_key,omitempty"`
	Organization    *OrganizationPublic `json:"organization,omitempty"`
}

// VerifyEventResponse is returned by Events.Verify.
type VerifyEventResponse struct {
	EventID       string              `json:"event_id"`
	MatchResult   bool                `json:"match_result"`
	MatchedField  *string             `json:"matched_field,omitempty"`
	AnchoredHash  string              `json:"anchored_hash"`
	SubmittedHash string              `json:"submitted_hash"`
	AnchoredAt    string              `json:"anchored_at"`
	Method        string              `json:"method"`
	Organization  *OrganizationPublic `json:"organization,omitempty"`
}

// ── Documents ───────────────────────────────────────────────

// AnchorDocumentResponse is returned by Documents.Anchor / AnchorFile.
type AnchorDocumentResponse struct {
	EventID      string `json:"event_id"`
	CreatedAt    string `json:"created_at"`
	DocumentHash string `json:"document_hash"`
	Status       string `json:"status"`
}

// DocumentListItem is one entry in a ListDocumentsResponse.
type DocumentListItem struct {
	EventID      string `json:"event_id"`
	DocumentRef  string `json:"document_ref"`
	DocumentHash string `json:"document_hash"`
	EventType    string `json:"event_type"`
	HasOriginal  bool   `json:"has_original"`
	CreatedAt    string `json:"created_at"`
}

// ListDocumentsResponse is a paginated document listing.
type ListDocumentsResponse struct {
	Documents []DocumentListItem `json:"documents"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
	Total     int                `json:"total"`
	HasMore   bool               `json:"has_more"`
}

// DocumentEvent is a single anchored document.
type DocumentEvent struct {
	EventID          string              `json:"event_id"`
	TenantID         string              `json:"tenant_id"`
	DocumentRef      string              `json:"document_ref"`
	DocumentHash     string              `json:"document_hash"`
	SignatureB64     string              `json:"signature_b64"`
	SignedPayloadB64 string              `json:"signed_payload_b64"`
	PublicKeyB64     string              `json:"public_key_b64"`
	HasOriginal      bool                `json:"has_original"`
	Metadata         map[string]any      `json:"metadata,omitempty"`
	CreatedAt        string              `json:"created_at"`
	Organization     *OrganizationPublic `json:"organization,omitempty"`
}

// VerifyDocumentResponse is returned by Documents.Verify.
type VerifyDocumentResponse struct {
	EventID       string              `json:"event_id"`
	MatchResult   bool                `json:"match_result"`
	DocumentRef   string              `json:"document_ref"`
	AnchoredHash  string              `json:"anchored_hash"`
	SubmittedHash string              `json:"submitted_hash"`
	AnchoredAt    string              `json:"anchored_at"`
	Organization  *OrganizationPublic `json:"organization,omitempty"`
}

// ── AI Attestations ─────────────────────────────────────────

// IngestAttestationResponse is returned by Attestations.Ingest.
type IngestAttestationResponse struct {
	AttestationID string `json:"attestation_id"`
	CreatedAt     string `json:"created_at"`
	InputHash     string `json:"input_hash"`
	OutputHash    string `json:"output_hash"`
	PayloadHash   string `json:"payload_hash"`
	Status        string `json:"status"`
}

// AttestationListItem is one entry in a ListAttestationsResponse.
type AttestationListItem struct {
	AttestationID   string  `json:"attestation_id"`
	AttestationType string  `json:"attestation_type"`
	AttestationHash string  `json:"attestation_hash"`
	ModelProvider   *string `json:"model_provider,omitempty"`
	ModelName       *string `json:"model_name,omitempty"`
	RetentionPolicy string  `json:"retention_policy"`
	CreatedAt       string  `json:"created_at"`
}

// ListAttestationsResponse is a paginated attestation listing.
type ListAttestationsResponse struct {
	Attestations []AttestationListItem `json:"attestations"`
	Page         int                   `json:"page"`
	Limit        int                   `json:"limit"`
	Total        int                   `json:"total"`
	HasMore      bool                  `json:"has_more"`
}

// AiAttestation is a single AI attestation record.
type AiAttestation struct {
	AttestationID   string              `json:"attestation_id"`
	TenantID        string              `json:"tenant_id"`
	AttestationType string              `json:"attestation_type"`
	AttestationHash string              `json:"attestation_hash"`
	InputHash       *string             `json:"input_hash,omitempty"`
	OutputHash      *string             `json:"output_hash,omitempty"`
	SignedPayload   string              `json:"signed_payload"`
	Signature       string              `json:"signature"`
	PublicKey       string              `json:"public_key"`
	SignatureAlg    string              `json:"signature_alg"`
	ModelProvider   *string             `json:"model_provider,omitempty"`
	ModelName       *string             `json:"model_name,omitempty"`
	ModelVersion    *string             `json:"model_version,omitempty"`
	RetentionPolicy string              `json:"retention_policy"`
	CreatedAt       string              `json:"created_at"`
	Organization    *OrganizationPublic `json:"organization,omitempty"`
}

// SignatureVerificationResult is the result of client-side Ed25519 signature
// verification via Attestations.VerifySignature.
type SignatureVerificationResult struct {
	// Valid reports whether the signature is valid.
	Valid bool
	// Reason is a human-readable reason when invalid; empty when valid.
	Reason string
	// Attestation is the record that was verified.
	Attestation AiAttestation
	// SignedData is the parsed JSON covered by the signature, or nil.
	SignedData map[string]any
}

// VerifyAttestationResponse is returned by Attestations.Verify.
type VerifyAttestationResponse struct {
	AttestationID string              `json:"attestation_id"`
	MatchResult   bool                `json:"match_result"`
	MatchedField  *string             `json:"matched_field,omitempty"`
	AnchoredHash  string              `json:"anchored_hash"`
	SubmittedHash string              `json:"submitted_hash"`
	AnchoredAt    string              `json:"anchored_at"`
	Organization  *OrganizationPublic `json:"organization,omitempty"`
}

// ── Traces ──────────────────────────────────────────────────

// CreateTraceResponse is returned by Traces.Create.
type CreateTraceResponse struct {
	TraceID   string `json:"trace_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Label     string `json:"label"`
}

// TraceListItem is one entry in a ListTracesResponse.
type TraceListItem struct {
	TraceID       string  `json:"trace_id"`
	Label         string  `json:"label"`
	Status        string  `json:"status"`
	EventCount    *int    `json:"event_count"`
	CreatedAt     string  `json:"created_at"`
	SealedAt      *string `json:"sealed_at"`
	CompositeHash *string `json:"composite_hash"`
}

// ListTracesResponse is a paginated trace listing.
type ListTracesResponse struct {
	Traces  []TraceListItem `json:"traces"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
	Total   int             `json:"total"`
	HasMore bool            `json:"has_more"`
}

// TraceEventSummary is a summary of an event in a trace.
type TraceEventSummary struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	PayloadHash string `json:"payload_hash"`
	IngestedAt  string `json:"ingested_at"`
}

// TraceDetail is a trace with paginated event summaries.
type TraceDetail struct {
	TraceID       string              `json:"trace_id"`
	Label         string              `json:"label"`
	Status        string              `json:"status"`
	EventCount    *int                `json:"event_count"`
	CreatedAt     string              `json:"created_at"`
	SealedAt      *string             `json:"sealed_at"`
	CompositeHash *string             `json:"composite_hash"`
	SealEventID   *string             `json:"seal_event_id"`
	Metadata      map[string]any      `json:"metadata,omitempty"`
	Events        []TraceEventSummary `json:"events"`
	EventPage     int                 `json:"event_page"`
	EventLimit    int                 `json:"event_limit"`
	EventTotal    int                 `json:"event_total"`
	EventHasMore  bool                `json:"event_has_more"`
}

// DeleteTraceResponse is returned by Traces.Delete.
type DeleteTraceResponse struct {
	TraceID string `json:"trace_id"`
	Deleted bool   `json:"deleted"`
}

// SealTraceResponse is returned by Traces.Seal.
type SealTraceResponse struct {
	TraceID string `json:"trace_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TraceProofEvent is a full event within a proof bundle.
type TraceProofEvent struct {
	EventID     string         `json:"event_id"`
	EventType   string         `json:"event_type"`
	Payload     map[string]any `json:"payload"`
	ContentHash string         `json:"content_hash"`
	Timestamp   string         `json:"timestamp"`
	Signature   string         `json:"signature"`
	PublicKey   string         `json:"public_key"`
}

// TraceProofSealEvent is the seal event within a proof bundle.
type TraceProofSealEvent struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	ContentHash string `json:"content_hash"`
	Timestamp   string `json:"timestamp"`
	Signature   string `json:"signature"`
	PublicKey   string `json:"public_key"`
}

// TraceProofVerification carries the server's verification summary.
type TraceProofVerification struct {
	CompositeHashValid bool `json:"composite_hash_valid"`
	AllSignaturesValid bool `json:"all_signatures_valid"`
}

// TraceProofBundle is an exported proof bundle for a sealed trace.
type TraceProofBundle struct {
	Version       string                 `json:"version"`
	TraceID       string                 `json:"trace_id"`
	Label         string                 `json:"label"`
	TenantDomain  string                 `json:"tenant_domain"`
	Status        string                 `json:"status"`
	CreatedAt     string                 `json:"created_at"`
	SealedAt      string                 `json:"sealed_at"`
	CompositeHash string                 `json:"composite_hash"`
	EventCount    int                    `json:"event_count"`
	Events        []TraceProofEvent      `json:"events"`
	SealEvent     TraceProofSealEvent    `json:"seal_event"`
	Verification  TraceProofVerification `json:"verification"`
}

// ── Audit Logs ──────────────────────────────────────────────

// AuditEvent is a single signed audit-ledger event. Unknown fields are not
// captured; use the map-returning audit methods for full fidelity.
type AuditEvent struct {
	ID               string         `json:"id"`
	OrgID            string         `json:"org_id"`
	Seq              int64          `json:"seq"`
	OccurredAt       string         `json:"occurred_at"`
	IngestedAt       string         `json:"ingested_at"`
	Action           string         `json:"action"`
	Actor            map[string]any `json:"actor"`
	Targets          []any          `json:"targets"`
	Context          map[string]any `json:"context,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	PayloadHash      string         `json:"payload_hash"`
	Signature        string         `json:"signature"`
	SigningPublicKey string         `json:"signing_public_key"`
}

// ListAuditEventsResponse is a keyset-paginated audit listing.
type ListAuditEventsResponse struct {
	Events     []AuditEvent `json:"events"`
	NextCursor *string      `json:"next_cursor"`
}
