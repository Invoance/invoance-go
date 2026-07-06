// Package invoance is the official Go SDK for the Invoance compliance API —
// cryptographic proof, document anchoring, AI attestation, traces, and audit
// logs.
//
// Create a client with New, reading INVOANCE_API_KEY (and optionally
// INVOANCE_BASE_URL) from the environment, or pass options explicitly:
//
//	client, err := invoance.New(invoance.WithAPIKey("inv_live_..."))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	resp, err := client.Events.Ingest(ctx, invoance.IngestEventParams{
//	    EventType: "user.login",
//	    Payload:   map[string]any{"user_id": "u_42"},
//	})
//
// Every network method takes a context.Context as its first argument and
// returns (T, error). All errors are *Error; branch on Kind or use the Is*
// predicate helpers.
package invoance

import "context"

// Client is the top-level SDK entry point. Construct it with New. Resource
// accessors are exposed as fields.
type Client struct {
	// Events is the compliance-events resource.
	Events *EventsResource
	// Documents is the document-anchoring resource.
	Documents *DocumentsResource
	// Attestations is the AI-attestations resource.
	Attestations *AttestationsResource
	// Traces is the traces resource.
	Traces *TracesResource
	// Audit is the audit-logs resource (with sub-resources).
	Audit *AuditResource

	transport *transport
	cfg       config
}

// New constructs a Client from functional options and the environment.
// INVOANCE_API_KEY is required unless WithAPIKey is passed; New returns an
// error (of kind Validation) when no API key is available.
func New(opts ...Option) (*Client, error) {
	cfg, err := resolveConfig(opts...)
	if err != nil {
		return nil, err
	}
	t := newTransport(cfg)
	c := &Client{
		transport: t,
		cfg:       cfg,
	}
	c.Events = &EventsResource{t: t}
	c.Documents = &DocumentsResource{t: t}
	c.Attestations = &AttestationsResource{t: t}
	c.Traces = &TracesResource{t: t}
	c.Audit = newAuditResource(t)
	return c, nil
}

// ValidationResult is the outcome of Client.Validate.
//
// Valid == true means the API key was accepted by the server (2xx, 403, or
// 429 — 403 and 429 still prove the key authenticated). Valid == false means
// the key was rejected, or the request never reached the server.
type ValidationResult struct {
	Valid   bool
	Reason  string
	BaseURL string
}

// Validate probes GET /v1/events?limit=1 to confirm the API key works. It
// never returns an error for expected outcomes — every failure mode is folded
// into the ValidationResult.
func (c *Client) Validate(ctx context.Context) ValidationResult {
	baseURL := c.cfg.baseURL
	limit := 1
	_, err := c.Events.List(ctx, ListEventsParams{Limit: &limit})
	if err == nil {
		return ValidationResult{Valid: true, Reason: "", BaseURL: baseURL}
	}
	switch {
	case IsAuthentication(err):
		return ValidationResult{Valid: false, Reason: "Authentication failed — check INVOANCE_API_KEY", BaseURL: baseURL}
	case IsForbidden(err):
		return ValidationResult{Valid: true, Reason: "API key authenticated but lacks permission to list events", BaseURL: baseURL}
	case IsQuotaExceeded(err):
		return ValidationResult{Valid: true, Reason: "API key authenticated but currently rate limited", BaseURL: baseURL}
	case IsNetwork(err) || IsTimeout(err):
		return ValidationResult{Valid: false, Reason: "Server unreachable: " + err.Error(), BaseURL: baseURL}
	default:
		return ValidationResult{Valid: false, Reason: err.Error(), BaseURL: baseURL}
	}
}
