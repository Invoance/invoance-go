package invoance

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

// Sign the golden event's canonical bytes with a freshly generated key and
// verify it back through the SDK's VerifyAuditEvent path.
func TestVerifyAuditEventRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	ev := goldenEvent()
	canonical, err := CanonicalAuditBytes(ev)
	if err != nil {
		t.Fatalf("CanonicalAuditBytes: %v", err)
	}
	sig := ed25519.Sign(priv, canonical)

	// Attach signature, key, and payload_hash the way the server would.
	ev["signature"] = hex.EncodeToString(sig)
	ev["signing_public_key"] = hex.EncodeToString(pub)
	ev["payload_hash"] = PayloadHashHex(canonical)

	// 1. Verify against embedded key.
	res := VerifyAuditEvent(ev, nil)
	if !res.Valid {
		t.Fatalf("embedded-key verify failed: %s", res.Reason)
	}
	if res.KeySource != KeySourceEvent {
		t.Errorf("keySource = %q, want event", res.KeySource)
	}
	if res.PayloadHash != PayloadHashHex(canonical) {
		t.Errorf("unexpected payload hash %s", res.PayloadHash)
	}

	// 2. Verify against a pinned key (hex string).
	pinned := VerifyAuditEvent(ev, &AuditVerifyOptions{PublicKey: hex.EncodeToString(pub)})
	if !pinned.Valid {
		t.Fatalf("pinned-key verify failed: %s", pinned.Reason)
	}
	if pinned.KeySource != KeySourcePinned {
		t.Errorf("keySource = %q, want pinned", pinned.KeySource)
	}

	// 3. Pinned key as raw bytes.
	pinnedBytes := VerifyAuditEvent(ev, &AuditVerifyOptions{PublicKey: []byte(pub)})
	if !pinnedBytes.Valid {
		t.Fatalf("pinned raw-bytes verify failed: %s", pinnedBytes.Reason)
	}

	// 4. Tamper with the signature -> invalid.
	tampered := map[string]any{}
	for k, v := range ev {
		tampered[k] = v
	}
	badSig := make([]byte, len(sig))
	copy(badSig, sig)
	badSig[0] ^= 0xFF
	tampered["signature"] = hex.EncodeToString(badSig)
	bad := VerifyAuditEvent(tampered, nil)
	if bad.Valid || bad.Reason != "signature_invalid" {
		t.Errorf("tampered signature: valid=%v reason=%q", bad.Valid, bad.Reason)
	}

	// 5. payload_hash mismatch -> invalid before signature check.
	mism := map[string]any{}
	for k, v := range ev {
		mism[k] = v
	}
	mism["payload_hash"] = "00" + PayloadHashHex(canonical)[2:]
	mres := VerifyAuditEvent(mism, nil)
	if mres.Valid || mres.Reason != "payload_hash_mismatch" {
		t.Errorf("mismatch: valid=%v reason=%q", mres.Valid, mres.Reason)
	}
}

func TestVerifyAuditEventNoKey(t *testing.T) {
	ev := goldenEvent()
	res := VerifyAuditEvent(ev, nil)
	if res.Valid || res.Reason != "no_public_key" {
		t.Errorf("expected no_public_key, got valid=%v reason=%q", res.Valid, res.Reason)
	}
}

func TestVerifyAuditEventStructWrapper(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	// Build via the map path to get a canonical + signature, then load the
	// typed struct with the same data.
	ev := goldenEvent()
	canonical, _ := CanonicalAuditBytes(ev)
	sig := ed25519.Sign(priv, canonical)

	seven := 7
	structEvent := AuditEvent{
		ID:               "aevt_456",
		OrgID:            "aorg_123",
		Seq:              int64(seven),
		IngestedAt:       "2026-01-02T03:04:05.6789Z",
		Action:           "user.login",
		OccurredAt:       "2026-01-02T04:04:05+01:00",
		Actor:            map[string]any{"type": "user", "id": "u_1"},
		Targets:          []any{map[string]any{"type": "doc", "id": "d_1"}},
		Metadata:         map[string]any{"z": "last", "a": "first"},
		PayloadHash:      PayloadHashHex(canonical),
		Signature:        hex.EncodeToString(sig),
		SigningPublicKey: hex.EncodeToString(pub),
	}
	res := VerifyAuditEventStruct(structEvent, nil)
	if !res.Valid {
		t.Fatalf("struct wrapper verify failed: %s", res.Reason)
	}
}
