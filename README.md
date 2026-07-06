# Invoance Go SDK

Official Go SDK for the [Invoance](https://invoance.com) compliance API —
cryptographic proof, document anchoring, AI attestation, traces, and audit
logs.

Zero non-stdlib dependencies. Synchronous, `context.Context`-aware, and
`(T, error)` throughout.

## Install

```bash
go get github.com/Invoance/invoance-go
```

Requires Go 1.21+.

## Quick start

Set your API key:

```bash
export INVOANCE_API_KEY=inv_live_...
```

```go
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Invoance/invoance-go"
)

func main() {
	ctx := context.Background()

	// Reads INVOANCE_API_KEY and INVOANCE_BASE_URL from the environment.
	client, err := invoance.New()
	if err != nil {
		log.Fatal(err)
	}

	// Ingest a compliance event.
	event, err := client.Events.Ingest(ctx, invoance.IngestEventParams{
		EventType: "policy.approval",
		Payload:   map[string]any{"policy_id": "pol_001", "decision": "approved"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(event.EventID)

	// Anchor a document by hash.
	docBytes := []byte("...your document bytes...")
	sum := sha256.Sum256(docBytes)
	doc, err := client.Documents.Anchor(ctx, invoance.AnchorDocumentParams{
		DocumentHash: hex.EncodeToString(sum[:]),
		DocumentRef:  "Invoice #1042",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(doc.EventID)

	// Or use the file helper (hashes + uploads in one call).
	anchored, err := client.Documents.AnchorFile(ctx, invoance.AnchorFileParams{
		Path:        "./invoice.pdf",
		DocumentRef: "Invoice #1042",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(anchored.EventID)

	// Ingest an AI attestation.
	att, err := client.Attestations.Ingest(ctx, invoance.IngestAttestationParams{
		Type:          "output",
		Input:         "Summarize this contract",
		Output:        "The contract states...",
		ModelProvider: "openai",
		ModelName:     "gpt-4o",
		ModelVersion:  "2025-01-01",
		Subject:       &invoance.AttestationSubject{UserID: "u_42", SessionID: "sess_4f9a"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(att.AttestationID)
}
```

## Quick validation

Sanity-check that your API key works before wiring the SDK into a larger app:

```go
client, _ := invoance.New()
res := client.Validate(context.Background())
if !res.Valid {
	log.Fatalf("Invoance: %s (base: %s)", res.Reason, res.BaseURL)
}
```

`Validate` probes `GET /v1/events?limit=1`, never returns an error, and
returns a `ValidationResult{Valid, Reason, BaseURL}` — use it in health checks,
startup scripts, or CI guards.

One-liner for a terminal sanity check, no SDK required:

```bash
curl -sS -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer $INVOANCE_API_KEY" \
  "${INVOANCE_BASE_URL:-https://api.invoance.com}/v1/events?limit=1"
# 200 = key valid · 401 = bad key · anything else = investigate
```

## Configuration

The client reads from environment variables automatically:

| Variable | Required | Default |
|---|---|---|
| `INVOANCE_API_KEY` | Yes | — |
| `INVOANCE_BASE_URL` | No | `https://api.invoance.com` |

You can also pass options explicitly:

```go
client, err := invoance.New(
	invoance.WithAPIKey("inv_live_..."),
	invoance.WithTimeout(60*time.Second),
	invoance.WithExtraHeaders(map[string]string{"X-Trace": "abc"}),
)
```

Available options: `WithAPIKey`, `WithBaseURL`, `WithAPIVersion`,
`WithTimeout`, `WithHTTPClient`, `WithIdempotencyKey`, `WithExtraHeaders`.

## Error handling

Every method returns an `error` that is always a `*invoance.Error`. Branch on
its `Kind`, or use the `Is*` predicate helpers (they walk the error chain via
`errors.As`):

```go
resp, err := client.Events.Ingest(ctx, params)
if err != nil {
	switch {
	case invoance.IsAuthentication(err):
		// 401 — bad API key
	case invoance.IsQuotaExceeded(err):
		var e *invoance.Error
		errors.As(err, &e)
		if e.RetryAfterSeconds != nil {
			log.Printf("rate limited, retry in %.0fs", *e.RetryAfterSeconds)
		}
	case invoance.IsValidation(err):
		// 400 or client-side validation
	case invoance.IsTimeout(err) || invoance.IsNetwork(err):
		// transport failure
	default:
		log.Printf("invoance error: %v", err)
	}
}
```

`Kind` values: `KindAuthentication`, `KindForbidden`, `KindNotFound`,
`KindValidation`, `KindConflict`, `KindQuotaExceeded`, `KindServer`,
`KindNetwork`, `KindTimeout`, `KindUnknown`.

## Resources

Every network method takes `ctx context.Context` as its first argument and
returns `(T, error)`. Optional scalar params are plain zero-omittable fields
(empty string / nil pointer = "omit"); flexible JSON blobs are `map[string]any`.

### Events

```go
// Ingest a compliance event (POST /events). EventType + Payload required;
// EventTime, TraceID, IdempotencyKey optional.
res, _ := client.Events.Ingest(ctx, invoance.IngestEventParams{
	EventType: "policy.approval",
	Payload:   map[string]any{"policy_id": "pol_001", "decision": "approved"},
	EventTime: "2026-07-06T12:00:00Z", // optional RFC3339
	TraceID:   "trc_123",              // optional
})
fmt.Println(res.EventID, res.IngestedAt)

// List events (GET /events). All fields optional; Page/Limit are *int.
page, limit := 1, 50
list, _ := client.Events.List(ctx, invoance.ListEventsParams{
	Page:      &page,
	Limit:     &limit,
	DateFrom:  "2026-01-01",
	DateTo:    "2026-07-01",
	EventType: "policy.approval",
})
fmt.Println(list.Total, list.HasMore)

// Get one event (GET /events/{id}).
event, _ := client.Events.Get(ctx, "evt_123")
fmt.Println(event.PayloadHash)

// Verify a hash or payload against the anchored event
// (POST /events/{id}/verify). Provide exactly one of PayloadHash or Payload.
v, _ := client.Events.Verify(ctx, "evt_123", invoance.VerifyEventParams{
	PayloadHash: "e3b0c44298fc1c149afbf4c8996fb924...", // 64-char hex SHA-256
})
// ...or let the server canonicalize + hash a raw payload:
v, _ = client.Events.Verify(ctx, "evt_123", invoance.VerifyEventParams{
	Payload: map[string]any{"policy_id": "pol_001", "decision": "approved"},
})
fmt.Println(v.MatchResult)
```

### Documents

```go
// Anchor a document by hash (POST /document/anchor). DocumentHash required
// (validated client-side); the rest optional.
doc, _ := client.Documents.Anchor(ctx, invoance.AnchorDocumentParams{
	DocumentHash:     hash, // 64-char hex SHA-256
	DocumentRef:      "Invoice #1042",
	EventType:        "invoice.issued",
	OriginalBytesB64: b64Bytes,                     // optional: upload the original
	Metadata:         map[string]any{"amount": 1200},
	TraceID:          "trc_123",
})
fmt.Println(doc.EventID, doc.Status)

// Convenience: hash + base64 + anchor in one call. Provide exactly one of
// Path or Bytes. DocumentRef defaults to the file's basename when Path is set.
anchored, _ := client.Documents.AnchorFile(ctx, invoance.AnchorFileParams{
	Path:        "./invoice.pdf",
	DocumentRef: "Invoice #1042",
})
// Raw bytes instead of a path; SkipOriginal to anchor the hash only:
anchored, _ = client.Documents.AnchorFile(ctx, invoance.AnchorFileParams{
	Bytes:        blob,
	DocumentRef:  "blob",
	SkipOriginal: true,
})
fmt.Println(anchored.EventID)

// List documents (GET /document).
docs, _ := client.Documents.List(ctx, invoance.ListDocumentsParams{
	DocumentRef: "Invoice #1042",
})
fmt.Println(docs.Total)

// Get one document's metadata (GET /document/{id}).
d, _ := client.Documents.Get(ctx, "evt_123")
fmt.Println(d.DocumentHash, d.HasOriginal)

// Download the original bytes (GET /document/{id}/original) => []byte.
raw, _ := client.Documents.GetOriginal(ctx, "evt_123")
_ = os.WriteFile("out.pdf", raw, 0o644)

// Verify a hash against the anchored document (POST /document/{id}/verify).
vd, _ := client.Documents.Verify(ctx, "evt_123", invoance.VerifyDocumentParams{
	DocumentHash: hash,
})
fmt.Println(vd.MatchResult)
```

### AI Attestations

```go
// Ingest an attestation (POST /ai/attestations). Type, Input, Output,
// ModelProvider, ModelName, ModelVersion required; Subject/TraceID optional.
att, _ := client.Attestations.Ingest(ctx, invoance.IngestAttestationParams{
	Type:          "output",
	Input:         "Summarize this contract",
	Output:        "The contract states...",
	ModelProvider: "openai",
	ModelName:     "gpt-4o",
	ModelVersion:  "2025-01-01",
	Subject: &invoance.AttestationSubject{
		UserID:    "u_42",
		SessionID: "sess_4f9a",
		Extra:     map[string]any{"tenant": "acme"}, // merged verbatim
	},
})
fmt.Println(att.AttestationID, att.PayloadHash)

// List attestations (GET /ai/attestations).
alist, _ := client.Attestations.List(ctx, invoance.ListAttestationsParams{
	AttestationType: "output",
	ModelProvider:   "openai",
})
fmt.Println(alist.Total)

// Get one attestation (GET /ai/attestations/{id}).
a, _ := client.Attestations.Get(ctx, "att_123")
fmt.Println(a.AttestationHash)

// Get the original canonical JSON as an untyped map
// (GET /ai/attestations/{id}/raw).
rawMap, _ := client.Attestations.GetRaw(ctx, "att_123")
fmt.Println(rawMap["type"])

// Server-side verify by content hash (POST /ai/attestations/{id}/verify).
va, _ := client.Attestations.Verify(ctx, "att_123", invoance.VerifyAttestationParams{
	ContentHash: "e3b0c44298fc1c149afbf4c8996fb924...", // 64-char hex
})
fmt.Println(va.MatchResult)

// Hash a raw payload client-side then verify. Pass the canonical JSON exactly
// as stored (see the note below) — []byte or string, key order preserved.
vp, _ := client.Attestations.VerifyPayload(ctx, "att_123",
	[]byte(`{"type":"output","payload":{...},"context":{...}}`))
vp, _ = client.Attestations.VerifyPayloadString(ctx, "att_123", rawJSON)
fmt.Println(vp.MatchResult)

// Verify the Ed25519 signature fully offline (fetches the record, verifies
// against its embedded 32-byte public key).
sig, _ := client.Attestations.VerifySignature(ctx, "att_123")
fmt.Println(sig.Valid, sig.Reason)
```

> **Note:** for `VerifyPayload` / `VerifyPayloadString`, pass the raw JSON
> **string or bytes** exactly as shown in the dashboard's "Raw immutable
> record" viewer. Key order is preserved (not sorted) because the backend
> hashes with serde struct field order (`type`, `payload`, `context`,
> `subject`). Do not unmarshal into a map first — that would lose order.

### Traces

```go
// Create a trace (POST /traces). Label required; Metadata optional.
trace, _ := client.Traces.Create(ctx, invoance.CreateTraceParams{
	Label:    "Batch 2026-07",
	Metadata: map[string]any{"customer": "acme"},
})
fmt.Println(trace.TraceID, trace.Status)

// List traces (GET /traces). Status filters "open" / "sealed".
tlist, _ := client.Traces.List(ctx, invoance.ListTracesParams{Status: "open"})
fmt.Println(tlist.Total)

// Get trace detail with paginated event summaries (GET /traces/{id}).
page, lim := 1, 50
detail, _ := client.Traces.Get(ctx, "trc_123", invoance.GetTraceParams{
	EventPage:  &page,
	EventLimit: &lim,
})
fmt.Println(detail.EventTotal, detail.CompositeHash)

// Seal a trace asynchronously (POST /traces/{id}/seal); server responds 202.
sealed, _ := client.Traces.Seal(ctx, "trc_123")
fmt.Println(sealed.Status, sealed.Message)

// Export the proof bundle as JSON, sealed traces only (GET /traces/{id}/proof).
proof, _ := client.Traces.Proof(ctx, "trc_123")
fmt.Println(proof.CompositeHash, proof.Verification.AllSignaturesValid)

// Download the proof bundle as a PDF (GET /traces/{id}/proof/pdf) => []byte.
pdf, _ := client.Traces.ProofPDF(ctx, "trc_123")
_ = os.WriteFile("proof.pdf", pdf, 0o644)

// Delete an empty, open trace (DELETE /traces/{id}).
del, _ := client.Traces.Delete(ctx, "trc_123")
fmt.Println(del.Deleted)
```

### Audit logs

The audit resource has five sub-resources: `Events`, `Orgs`, `Streams`,
`PortalSessions`, and `Exports`. The audit ledger requires an
`Idempotency-Key`; if you don't supply one on `Events.Ingest`, the SDK derives
a content-stable key from the request body for safe retries.

**`client.Audit.Events`** — the signed audit-event ledger:

```go
// Append one signed event (POST /audit/events). OrganizationID, Action, Actor
// required; OccurredAt defaults to now, Targets to []. IdempotencyKey overrides
// the derived content key.
ev, _ := client.Audit.Events.Ingest(ctx, invoance.IngestAuditEventParams{
	OrganizationID: "org_acme",
	Action:         "invoice.approved",
	Actor:          map[string]any{"type": "user", "id": "u_1"},
	Targets:        []map[string]any{{"type": "invoice", "id": "inv_42"}},
	Context:        map[string]any{"ip": "1.2.3.4"},
	Metadata:       map[string]any{"amount": 1200},
})

// Keyset-paginated listing (GET /audit/events).
limit := 100
alist, _ := client.Audit.Events.List(ctx, invoance.ListAuditEventsParams{
	OrganizationID: "org_acme",
	Actions:        "invoice.approved",
	ActorID:        "u_1",
	RangeStart:     "2026-01-01T00:00:00Z",
	RangeEnd:       "2026-07-01T00:00:00Z",
	Limit:          &limit,
	Cursor:         "", // pass NextCursor from the prior page
})
fmt.Println(len(alist.Events), alist.NextCursor)

// Get one audit event (GET /audit/events/{id}) => typed AuditEvent.
one, _ := client.Audit.Events.Get(ctx, "aevt_1")

// Server-side verify with the pinned key (GET /audit/events/{id}/verify).
sv, _ := client.Audit.Events.Verify(ctx, "aevt_1")
fmt.Println(sv["valid"])
```

**`client.Audit.Orgs`** — end-customer orgs:

```go
_, _ = client.Audit.Orgs.Create(ctx, invoance.CreateAuditOrgParams{
	OrganizationID: "org_acme",
	Name:           "Acme Inc.", // optional
})
orgs, _ := client.Audit.Orgs.List(ctx)                     // GET /audit/orgs
rep, _ := client.Audit.Orgs.Integrity(ctx, "org_acme")     // integrity report
_, _ = client.Audit.Orgs.SetRetention(ctx, "org_acme", 365) // retention days
_ = orgs
_ = rep
```

**`client.Audit.Streams`** — SIEM/webhook delivery (org-scoped; the signing
secret is returned once on create):

```go
s, _ := client.Audit.Streams.Create(ctx, "org_acme", invoance.CreateAuditStreamParams{
	URL:  "https://siem.example/hook",
	Type: "webhook", // default; v1 supports webhook only
})
streams, _ := client.Audit.Streams.List(ctx, "org_acme")
_, _ = client.Audit.Streams.Test(ctx, "org_acme", "stream_1")   // test delivery
_, _ = client.Audit.Streams.Delete(ctx, "org_acme", "stream_1")
_ = s
_ = streams
```

**`client.Audit.PortalSessions`** — one-time hosted-viewer links:

```go
sessDur, linkDur := 3600, 600
portal, _ := client.Audit.PortalSessions.Create(ctx, invoance.CreatePortalSessionParams{
	OrganizationID:         "org_acme",
	Intent:                 "audit_logs", // "audit_logs" or "log_streams"
	SessionDurationSeconds: &sessDur,     // optional
	LinkDurationSeconds:    &linkDur,     // optional
})
fmt.Println(portal["url"])
```

**`client.Audit.Exports`** — async export jobs:

```go
job, _ := client.Audit.Exports.Create(ctx, invoance.CreateAuditExportParams{
	OrganizationID: "org_acme",
	Format:         "csv", // "csv" or "ndjson"
	Filters:        map[string]any{"action": "invoice.approved"}, // optional
})
// Poll until ready; a "ready" response carries a download_url.
status, _ := client.Audit.Exports.Get(ctx, job["export_id"].(string))
fmt.Println(status["status"], status["download_url"])
```

## Offline verification

The SDK can verify signatures and hashes entirely client-side — no trust in the
server required. See the per-resource sections above for `Attestations.
VerifySignature` / `VerifyPayload`; the audit-ledger equivalents are top-level
package functions.

```go
// Verify an attestation's Ed25519 signature.
res, _ := client.Attestations.VerifySignature(ctx, "att_123")
fmt.Println(res.Valid, res.Reason)

// Verify a typed audit event offline. Pass nil opts to trust the event's
// embedded key, or pin the tenant's registered key for a real tamper guarantee.
event, _ := client.Audit.Events.Get(ctx, "aevt_1")
result := invoance.VerifyAuditEventStruct(event, nil)
fmt.Println(result.Valid, result.PayloadHash, result.KeySource)

// Pin the registered key (hex string or []byte):
result = invoance.VerifyAuditEventStruct(event, &invoance.AuditVerifyOptions{
	PublicKey: registeredHexKey,
})

// VerifyAuditEvent takes a raw map[string]any (e.g. from a map-returning audit
// method) for full fidelity when you have unknown fields.
raw, _ := client.Audit.Events.Verify(ctx, "aevt_1")
_ = invoance.VerifyAuditEvent(raw, nil)
```

For attestation payload verification, pass the raw JSON **string or bytes**
exactly as stored (the dashboard's "Raw immutable record"). Key order is
preserved — do not decode into a map first:

```go
raw := []byte(`{"type":"output","payload":{...},"context":{...}}`)
v, _ := client.Attestations.VerifyPayload(ctx, "att_123", raw)
fmt.Println(v.MatchResult)
```

## License

MIT © 2026 Invoance, Inc.
