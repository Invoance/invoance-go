package invoance

import "context"

// EventsResource is the compliance-events resource (client.Events).
type EventsResource struct {
	t *transport
}

// IngestEventParams are the parameters for Events.Ingest.
type IngestEventParams struct {
	// EventType is the required event classifier.
	EventType string
	// Payload is the required JSON object payload.
	Payload map[string]any
	// EventTime is an optional caller-supplied timestamp (RFC3339). Omitted
	// when empty.
	EventTime string
	// IdempotencyKey is an optional per-call idempotency key.
	IdempotencyKey string
	// TraceID optionally associates the event with a trace. Omitted when empty.
	TraceID string
}

// Ingest anchors a compliance event (POST /events).
func (r *EventsResource) Ingest(ctx context.Context, params IngestEventParams) (IngestEventResponse, error) {
	body := map[string]any{
		"event_type": params.EventType,
		"payload":    params.Payload,
	}
	if params.EventTime != "" {
		body["event_time"] = params.EventTime
	}
	if params.TraceID != "" {
		body["trace_id"] = params.TraceID
	}
	var out IngestEventResponse
	err := r.t.post(ctx, "/events", body, params.IdempotencyKey, &out)
	return out, err
}

// ListEventsParams are the query parameters for Events.List.
type ListEventsParams struct {
	Page      *int
	Limit     *int
	DateFrom  string
	DateTo    string
	EventType string
}

// List returns a paginated event listing (GET /events).
func (r *EventsResource) List(ctx context.Context, params ListEventsParams) (ListEventsResponse, error) {
	var out ListEventsResponse
	err := r.t.get(ctx, "/events", map[string]any{
		"page":       params.Page,
		"limit":      params.Limit,
		"date_from":  params.DateFrom,
		"date_to":    params.DateTo,
		"event_type": params.EventType,
	}, &out)
	return out, err
}

// Get retrieves a single event (GET /events/{eventID}).
func (r *EventsResource) Get(ctx context.Context, eventID string) (ComplianceEvent, error) {
	var out ComplianceEvent
	err := r.t.get(ctx, "/events/"+eventID, nil, &out)
	return out, err
}

// VerifyEventParams are the parameters for Events.Verify. Provide exactly one
// of PayloadHash (hex SHA-256) or Payload.
type VerifyEventParams struct {
	// PayloadHash is a 64-char lowercase hex SHA-256 digest.
	PayloadHash string
	// Payload is a raw JSON object the server canonicalizes and hashes.
	Payload map[string]any
}

// Verify checks a submitted hash or payload against the anchored event
// (POST /events/{eventID}/verify). Passing neither PayloadHash nor Payload
// returns a client-side validation error.
func (r *EventsResource) Verify(ctx context.Context, eventID string, params VerifyEventParams) (VerifyEventResponse, error) {
	var out VerifyEventResponse
	if params.PayloadHash == "" && params.Payload == nil {
		return out, &Error{
			Kind:    KindValidation,
			Message: "events.Verify requires either PayloadHash or Payload",
		}
	}
	body := map[string]any{}
	if params.PayloadHash != "" {
		if err := assertSha256Hex("payloadHash", params.PayloadHash); err != nil {
			return out, err
		}
		body["payload_hash"] = params.PayloadHash
	}
	if params.Payload != nil {
		body["payload"] = params.Payload
	}
	err := r.t.post(ctx, "/events/"+eventID+"/verify", body, "", &out)
	return out, err
}
