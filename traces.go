package invoance

import "context"

// TracesResource is the traces resource (client.Traces).
type TracesResource struct {
	t *transport
}

// CreateTraceParams are the parameters for Traces.Create.
type CreateTraceParams struct {
	// Label is the required human-readable trace label.
	Label string
	// Metadata is optional arbitrary JSON metadata.
	Metadata map[string]any
}

// Create creates a new trace (POST /traces).
func (r *TracesResource) Create(ctx context.Context, params CreateTraceParams) (CreateTraceResponse, error) {
	body := map[string]any{"label": params.Label}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}
	var out CreateTraceResponse
	err := r.t.post(ctx, "/traces", body, "", &out)
	return out, err
}

// ListTracesParams are the query parameters for Traces.List.
type ListTracesParams struct {
	Page  *int
	Limit *int
	// Status filters by "open" or "sealed"; empty means no filter.
	Status string
}

// List returns a paginated trace listing (GET /traces).
func (r *TracesResource) List(ctx context.Context, params ListTracesParams) (ListTracesResponse, error) {
	var out ListTracesResponse
	err := r.t.get(ctx, "/traces", map[string]any{
		"page":   params.Page,
		"limit":  params.Limit,
		"status": params.Status,
	}, &out)
	return out, err
}

// GetTraceParams are the event-pagination parameters for Traces.Get.
type GetTraceParams struct {
	// EventPage is the 1-based event page (default 1).
	EventPage *int
	// EventLimit is the max events per page (default 50, max 200).
	EventLimit *int
}

// Get retrieves trace detail with paginated events (GET /traces/{traceID}).
func (r *TracesResource) Get(ctx context.Context, traceID string, params GetTraceParams) (TraceDetail, error) {
	var out TraceDetail
	err := r.t.get(ctx, "/traces/"+traceID, map[string]any{
		"event_page":  params.EventPage,
		"event_limit": params.EventLimit,
	}, &out)
	return out, err
}

// Delete deletes an empty open trace (DELETE /traces/{traceID}).
func (r *TracesResource) Delete(ctx context.Context, traceID string) (DeleteTraceResponse, error) {
	var out DeleteTraceResponse
	err := r.t.delete(ctx, "/traces/"+traceID, &out)
	return out, err
}

// Seal seals a trace asynchronously (POST /traces/{traceID}/seal). The server
// responds with 202.
func (r *TracesResource) Seal(ctx context.Context, traceID string) (SealTraceResponse, error) {
	var out SealTraceResponse
	err := r.t.post(ctx, "/traces/"+traceID+"/seal", map[string]any{}, "", &out)
	return out, err
}

// Proof exports the proof bundle as JSON, for sealed traces only
// (GET /traces/{traceID}/proof).
func (r *TracesResource) Proof(ctx context.Context, traceID string) (TraceProofBundle, error) {
	var out TraceProofBundle
	err := r.t.get(ctx, "/traces/"+traceID+"/proof", nil, &out)
	return out, err
}

// ProofPDF downloads the proof bundle as a PDF, for sealed traces only
// (GET /traces/{traceID}/proof/pdf).
func (r *TracesResource) ProofPDF(ctx context.Context, traceID string) ([]byte, error) {
	return r.t.getBytes(ctx, "/traces/"+traceID+"/proof/pdf")
}
