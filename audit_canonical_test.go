package invoance

import (
	"testing"
)

// A known audit event and its expected canonical form. The event is
// deliberately given with keys out of order, with a null field to strip, and
// with a non-UTC timestamp to normalize, to exercise the full pipeline.
func goldenEvent() map[string]any {
	return map[string]any{
		"org_id":      "aorg_123",
		"event_id":    "aevt_456",
		"seq":         float64(7),
		"ingested_at": "2026-01-02T03:04:05.6789Z",
		"action":      "user.login",
		"occurred_at": "2026-01-02T04:04:05+01:00", // +01:00 -> 03:04:05Z
		"actor":       map[string]any{"type": "user", "id": "u_1"},
		"targets":     []any{map[string]any{"type": "doc", "id": "d_1"}},
		"metadata":    map[string]any{"z": "last", "a": "first", "drop": nil},
		// context intentionally omitted
	}
}

const goldenCanonical = `{"action":"user.login","actor":{"id":"u_1","type":"user"},"event_id":"aevt_456","ingested_at":"2026-01-02T03:04:05.678Z","metadata":{"a":"first","z":"last"},"occurred_at":"2026-01-02T03:04:05.000Z","org_id":"aorg_123","schema_id":"invoance.audit/1","seq":7,"targets":[{"id":"d_1","type":"doc"}]}`

const goldenPayloadHash = "45cf57ccf8baca7f37d23fd01602895ae67913f6c77b27082acaf96f2a9c99ba"

func TestCanonicalAuditBytesGolden(t *testing.T) {
	got, err := CanonicalAuditBytes(goldenEvent())
	if err != nil {
		t.Fatalf("CanonicalAuditBytes error: %v", err)
	}
	if string(got) != goldenCanonical {
		t.Errorf("canonical bytes mismatch\n got: %s\nwant: %s", got, goldenCanonical)
	}
	if h := PayloadHashHex(got); h != goldenPayloadHash {
		t.Errorf("payload hash mismatch\n got: %s\nwant: %s", h, goldenPayloadHash)
	}
}

func TestNormalizeTS(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"2026-01-02T03:04:05Z", "2026-01-02T03:04:05.000Z"},
		{"2026-01-02T03:04:05.6789Z", "2026-01-02T03:04:05.678Z"}, // truncate, not round
		{"2026-01-02T04:04:05+01:00", "2026-01-02T03:04:05.000Z"},
		{"2026-01-02t03:04:05.1z", "2026-01-02T03:04:05.100Z"},
		{"2026-01-02T00:30:00-05:30", "2026-01-02T06:00:00.000Z"},
	}
	for _, c := range cases {
		got, err := NormalizeTS(c.in)
		if err != nil {
			t.Errorf("NormalizeTS(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeTS(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if _, err := NormalizeTS("not-a-timestamp"); err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestBuildSignedObjectMissingRequired(t *testing.T) {
	ev := goldenEvent()
	delete(ev, "action")
	if _, err := CanonicalAuditBytes(ev); err == nil {
		t.Error("expected error when a required field is missing")
	}
}
