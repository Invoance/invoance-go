package invoance

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// AttestationsResource is the AI-attestations resource (client.Attestations).
type AttestationsResource struct {
	t *transport
}

// IngestAttestationParams are the parameters for Attestations.Ingest.
type IngestAttestationParams struct {
	// Type is the required attestation type (e.g. "output").
	Type string
	// Input is the required model input string.
	Input string
	// Output is the required model output string.
	Output string
	// ModelProvider is the required provider (e.g. "openai").
	ModelProvider string
	// ModelName is the required model name (e.g. "gpt-4o").
	ModelName string
	// ModelVersion is the required model version.
	ModelVersion string
	// Subject optionally identifies who/what triggered the attestation.
	// UserID maps to user_id, SessionID to session_id; Extra keys pass
	// through verbatim. Omitted entirely when empty.
	Subject *AttestationSubject
	// IdempotencyKey is an optional per-call idempotency key.
	IdempotencyKey string
	// TraceID optionally associates the attestation with a trace.
	TraceID string
}

// AttestationSubject is the optional subject context for an attestation.
type AttestationSubject struct {
	// UserID maps to the "user_id" wire key.
	UserID string
	// SessionID maps to the "session_id" wire key.
	SessionID string
	// Extra holds additional tenant-specific keys, merged as-is.
	Extra map[string]any
}

// Ingest anchors an AI attestation (POST /ai/attestations).
func (r *AttestationsResource) Ingest(ctx context.Context, params IngestAttestationParams) (IngestAttestationResponse, error) {
	body := map[string]any{
		"type": params.Type,
		"payload": map[string]any{
			"input":  params.Input,
			"output": params.Output,
		},
		"context": map[string]any{
			"model_provider": params.ModelProvider,
			"model_name":     params.ModelName,
			"model_version":  params.ModelVersion,
		},
	}

	if params.Subject != nil {
		subject := map[string]any{}
		if params.Subject.UserID != "" {
			subject["user_id"] = params.Subject.UserID
		}
		if params.Subject.SessionID != "" {
			subject["session_id"] = params.Subject.SessionID
		}
		for k, v := range params.Subject.Extra {
			subject[k] = v
		}
		if len(subject) > 0 {
			body["subject"] = subject
		}
	}

	if params.TraceID != "" {
		body["trace_id"] = params.TraceID
	}

	var out IngestAttestationResponse
	err := r.t.post(ctx, "/ai/attestations", body, params.IdempotencyKey, &out)
	return out, err
}

// ListAttestationsParams are the query parameters for Attestations.List.
type ListAttestationsParams struct {
	Page            *int
	Limit           *int
	DateFrom        string
	DateTo          string
	AttestationType string
	ModelProvider   string
}

// List returns a paginated attestation listing (GET /ai/attestations).
func (r *AttestationsResource) List(ctx context.Context, params ListAttestationsParams) (ListAttestationsResponse, error) {
	var out ListAttestationsResponse
	err := r.t.get(ctx, "/ai/attestations", map[string]any{
		"page":             params.Page,
		"limit":            params.Limit,
		"date_from":        params.DateFrom,
		"date_to":          params.DateTo,
		"attestation_type": params.AttestationType,
		"model_provider":   params.ModelProvider,
	}, &out)
	return out, err
}

// Get retrieves a single attestation (GET /ai/attestations/{id}).
func (r *AttestationsResource) Get(ctx context.Context, attestationID string) (AiAttestation, error) {
	var out AiAttestation
	err := r.t.get(ctx, "/ai/attestations/"+attestationID, nil, &out)
	return out, err
}

// VerifyAttestationParams are the parameters for Attestations.Verify.
type VerifyAttestationParams struct {
	// ContentHash is the 64-char lowercase hex SHA-256 digest to check.
	ContentHash string
}

// Verify checks a submitted content hash against the anchored attestation
// (POST /ai/attestations/{id}/verify).
func (r *AttestationsResource) Verify(ctx context.Context, attestationID string, params VerifyAttestationParams) (VerifyAttestationResponse, error) {
	var out VerifyAttestationResponse
	if err := assertSha256Hex("contentHash", params.ContentHash); err != nil {
		return out, err
	}
	err := r.t.post(ctx, "/ai/attestations/"+attestationID+"/verify", map[string]any{
		"content_hash": params.ContentHash,
	}, "", &out)
	return out, err
}

// GetRaw retrieves the original canonical JSON payload as an untyped map
// (GET /ai/attestations/{id}/raw).
func (r *AttestationsResource) GetRaw(ctx context.Context, attestationID string) (map[string]any, error) {
	var out map[string]any
	err := r.t.getRaw(ctx, "/ai/attestations/"+attestationID+"/raw", &out)
	return out, err
}

// VerifyPayload hashes a raw payload client-side and calls Verify.
//
// The payload must be the canonical JSON stored in Invoance (the "Raw
// immutable record"). Pass it as a JSON string or []byte to PRESERVE KEY
// ORDER — the backend hashes with serde_json struct field order
// (type, payload, context, subject), NOT alphabetical order. The bytes are
// compacted (whitespace stripped) with encoding/json.Compact, which preserves
// order; they are NOT re-sorted. Do not unmarshal into a map first, as that
// would lose order.
func (r *AttestationsResource) VerifyPayload(ctx context.Context, attestationID string, payload []byte) (VerifyAttestationResponse, error) {
	var out VerifyAttestationResponse
	var compact bytes.Buffer
	if err := json.Compact(&compact, payload); err != nil {
		return out, &Error{Kind: KindValidation, Message: "verifyPayload: invalid JSON payload: " + err.Error(), cause: err}
	}
	sum := sha256.Sum256(compact.Bytes())
	contentHash := hex.EncodeToString(sum[:])
	return r.Verify(ctx, attestationID, VerifyAttestationParams{ContentHash: contentHash})
}

// VerifyPayloadString is a convenience wrapper over VerifyPayload for string
// input. String input is the safest choice for key-order fidelity.
func (r *AttestationsResource) VerifyPayloadString(ctx context.Context, attestationID string, payload string) (VerifyAttestationResponse, error) {
	return r.VerifyPayload(ctx, attestationID, []byte(payload))
}

// VerifySignature fetches the attestation and verifies its Ed25519 signature
// fully client-side, using the raw 32-byte public key embedded in the record.
func (r *AttestationsResource) VerifySignature(ctx context.Context, attestationID string) (SignatureVerificationResult, error) {
	att, err := r.Get(ctx, attestationID)
	if err != nil {
		return SignatureVerificationResult{}, err
	}

	result := SignatureVerificationResult{Attestation: att}

	signedPayloadBytes, err1 := hex.DecodeString(att.SignedPayload)
	signatureBytes, err2 := hex.DecodeString(att.Signature)
	publicKeyBytes, err3 := hex.DecodeString(att.PublicKey)

	switch {
	case err1 != nil:
		result.Reason = "signed_payload is not valid hex"
	case err2 != nil:
		result.Reason = "signature is not valid hex"
	case err3 != nil:
		result.Reason = "public_key is not valid hex"
	case len(publicKeyBytes) != ed25519.PublicKeySize:
		result.Reason = "public_key is not a 32-byte Ed25519 key"
	default:
		valid := ed25519.Verify(ed25519.PublicKey(publicKeyBytes), signedPayloadBytes, signatureBytes)
		result.Valid = valid
		if !valid {
			result.Reason = "Signature does not match signed_payload + public_key"
		}
	}

	// Parse the signed payload to show what was covered by the signature.
	if len(signedPayloadBytes) > 0 {
		var parsed map[string]any
		if json.Unmarshal(signedPayloadBytes, &parsed) == nil {
			result.SignedData = parsed
		}
	}

	return result, nil
}
