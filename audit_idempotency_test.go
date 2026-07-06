package invoance

import (
	"strings"
	"testing"
)

func TestContentIdempotencyKeyStability(t *testing.T) {
	// Two logically identical bodies with keys in different insertion order
	// must produce the same key (map iteration order is irrelevant; JSON
	// marshaling sorts keys).
	a := map[string]any{
		"organization_id": "aorg_1",
		"action":          "user.login",
		"actor":           map[string]any{"type": "user", "id": "u_1"},
		"targets":         []any{},
		"occurred_at":     "2026-01-02T03:04:05Z",
	}
	b := map[string]any{
		"actor":           map[string]any{"id": "u_1", "type": "user"},
		"action":          "user.login",
		"occurred_at":     "2026-01-02T03:04:05Z",
		"organization_id": "aorg_1",
		"targets":         []any{},
	}

	ka := ContentIdempotencyKey(a)
	kb := ContentIdempotencyKey(b)
	if ka != kb {
		t.Errorf("idempotency keys differ for equal bodies:\n a=%s\n b=%s", ka, kb)
	}
	if !strings.HasPrefix(ka, "idem_") {
		t.Errorf("key missing idem_ prefix: %s", ka)
	}
	if len(ka) != len("idem_")+64 {
		t.Errorf("key length = %d, want %d", len(ka), len("idem_")+64)
	}

	// Golden value (regenerated if the stable form ever changes).
	const golden = "idem_15e93199dbdbd289bc598fe526e7c9b134af8a0660e9644bcc6af6889b58a0ed"
	if ka != golden {
		t.Errorf("idempotency key golden mismatch\n got: %s\nwant: %s", ka, golden)
	}
}
