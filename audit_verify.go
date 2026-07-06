package invoance

import (
	"crypto/ed25519"
	"encoding/hex"
)

// AuditKeySource indicates which key was used to verify an audit event.
type AuditKeySource string

const (
	// KeySourcePinned means a caller-supplied public key was used.
	KeySourcePinned AuditKeySource = "pinned"
	// KeySourceEvent means the key embedded in the event was used.
	KeySourceEvent AuditKeySource = "event"
)

// AuditVerifyResult is the outcome of VerifyAuditEvent.
type AuditVerifyResult struct {
	// Valid reports whether the signature verified.
	Valid bool
	// Reason is a machine reason when invalid; empty when valid.
	Reason string
	// PayloadHash is the recomputed canonical payload hash (hex).
	PayloadHash string
	// KeySource indicates whether a pinned or embedded key was used.
	KeySource AuditKeySource
}

// AuditVerifyOptions configures VerifyAuditEvent.
type AuditVerifyOptions struct {
	// PublicKey optionally pins a trusted key (hex or raw bytes). When set,
	// the event's embedded key is ignored and KeySource is "pinned".
	PublicKey any
}

// VerifyAuditEvent verifies one audit event's Ed25519 signature offline,
// against either a pinned public key or the key embedded in the event.
//
// The event is a decoded JSON object (map[string]any) — pass the raw map from
// audit list/get responses for full fidelity, since it reads event["id"] or
// event["event_id"]. See VerifyAuditEventStruct for verifying an AuditEvent.
//
// Trust note: verifying against the event-embedded key only proves internal
// consistency. For a real tamper guarantee, pin the tenant's registered key
// via AuditVerifyOptions.PublicKey.
func VerifyAuditEvent(event map[string]any, opts *AuditVerifyOptions) AuditVerifyResult {
	keySource := KeySourceEvent
	if opts != nil && opts.PublicKey != nil {
		keySource = KeySourcePinned
	}

	eventID := event["event_id"]
	if v, ok := event["id"]; ok && v != nil {
		eventID = v
	}

	signedInput := map[string]any{
		"org_id":      event["org_id"],
		"event_id":    eventID,
		"seq":         event["seq"],
		"ingested_at": event["ingested_at"],
		"action":      event["action"],
		"occurred_at": event["occurred_at"],
		"actor":       event["actor"],
		"targets":     event["targets"],
	}
	if v, ok := event["context"]; ok && v != nil {
		signedInput["context"] = v
	}
	if v, ok := event["metadata"]; ok && v != nil {
		signedInput["metadata"] = v
	}

	canonical, err := CanonicalAuditBytes(signedInput)
	if err != nil {
		return AuditVerifyResult{Valid: false, Reason: "canonicalization_failed", PayloadHash: "", KeySource: keySource}
	}

	recomputed := PayloadHashHex(canonical)
	if ph, ok := event["payload_hash"].(string); ok && ph != "" && ph != recomputed {
		return AuditVerifyResult{Valid: false, Reason: "payload_hash_mismatch", PayloadHash: recomputed, KeySource: keySource}
	}

	keyBytes, keyOK := resolveKeyBytes(opts, event)
	if !keyOK {
		return AuditVerifyResult{Valid: false, Reason: "no_public_key", PayloadHash: recomputed, KeySource: keySource}
	}

	sigHex, sigOK := event["signature"].(string)
	if !sigOK || sigHex == "" {
		return AuditVerifyResult{Valid: false, Reason: "no_signature", PayloadHash: recomputed, KeySource: keySource}
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return AuditVerifyResult{Valid: false, Reason: "signature_invalid", PayloadHash: recomputed, KeySource: keySource}
	}

	if len(keyBytes) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(keyBytes), canonical, sigBytes) {
		return AuditVerifyResult{Valid: false, Reason: "signature_invalid", PayloadHash: recomputed, KeySource: keySource}
	}
	return AuditVerifyResult{Valid: true, Reason: "", PayloadHash: recomputed, KeySource: keySource}
}

// resolveKeyBytes returns the public key bytes to verify with, from the pinned
// option (hex string or raw bytes) or the event's signing_public_key (hex).
func resolveKeyBytes(opts *AuditVerifyOptions, event map[string]any) ([]byte, bool) {
	if opts != nil && opts.PublicKey != nil {
		switch k := opts.PublicKey.(type) {
		case string:
			b, err := hex.DecodeString(k)
			if err != nil {
				return nil, false
			}
			return b, true
		case []byte:
			return k, true
		}
		return nil, false
	}
	if k, ok := event["signing_public_key"].(string); ok && k != "" {
		b, err := hex.DecodeString(k)
		if err != nil {
			return nil, false
		}
		return b, true
	}
	return nil, false
}

// VerifyAuditEventStruct is a typed convenience wrapper over VerifyAuditEvent
// for an AuditEvent value.
func VerifyAuditEventStruct(event AuditEvent, opts *AuditVerifyOptions) AuditVerifyResult {
	m := map[string]any{
		"id":                 event.ID,
		"org_id":             event.OrgID,
		"seq":                event.Seq,
		"occurred_at":        event.OccurredAt,
		"ingested_at":        event.IngestedAt,
		"action":             event.Action,
		"actor":              event.Actor,
		"targets":            toAnySlice(event.Targets),
		"payload_hash":       event.PayloadHash,
		"signature":          event.Signature,
		"signing_public_key": event.SigningPublicKey,
	}
	if event.Context != nil {
		m["context"] = event.Context
	}
	if event.Metadata != nil {
		m["metadata"] = event.Metadata
	}
	return VerifyAuditEvent(m, opts)
}

func toAnySlice(v []any) any {
	if v == nil {
		return nil
	}
	return v
}
