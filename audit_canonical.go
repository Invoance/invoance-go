package invoance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// AuditSchemaID is the frozen canonicalization schema identifier.
const AuditSchemaID = "invoance.audit/1"

// signedFields is the ordered set of fields the canonicalizer considers.
var signedFields = []string{
	"org_id", "event_id", "seq", "ingested_at", "action",
	"occurred_at", "actor", "targets", "context", "metadata",
}

// requiredFields must be present and non-nil for canonicalization to succeed.
var requiredFields = []string{
	"org_id", "event_id", "seq", "ingested_at", "action",
	"occurred_at", "actor", "targets",
}

var rfc3339Regexp = regexp.MustCompile(
	`^(\d{4})-(\d{2})-(\d{2})[Tt](\d{2}):(\d{2}):(\d{2})(?:\.(\d+))?(Z|z|[+-]\d{2}:\d{2})$`)

// NormalizeTS converts an RFC3339 timestamp to the one canonical form (§4.4):
// UTC, exactly 3 fractional digits (TRUNCATED, not rounded), suffix "Z".
func NormalizeTS(value string) (string, error) {
	m := rfc3339Regexp.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return "", fmt.Errorf("invalid RFC3339 timestamp: %s", value)
	}
	yr, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	dy, _ := strconv.Atoi(m[3])
	hh, _ := strconv.Atoi(m[4])
	mi, _ := strconv.Atoi(m[5])
	ss, _ := strconv.Atoi(m[6])

	// Truncate fractional seconds to exactly 3 digits (milliseconds).
	fracStr := (m[7] + "000")[:3]
	millis, _ := strconv.Atoi(fracStr)

	off := m[8]
	// Build UTC time, subtracting the offset to normalize to UTC.
	t := time.Date(yr, time.Month(mo), dy, hh, mi, ss, millis*int(time.Millisecond), time.UTC)
	if off != "Z" && off != "z" {
		sign := 1
		if off[0] == '-' {
			sign = -1
		}
		oh, _ := strconv.Atoi(off[1:3])
		om, _ := strconv.Atoi(off[4:6])
		t = t.Add(time.Duration(-sign) * (time.Duration(oh)*time.Hour + time.Duration(om)*time.Minute))
	}
	return t.Format("2006-01-02T15:04:05.000Z"), nil
}

// buildSignedObject builds the signed object: required fields present &
// non-nil, timestamps normalized, forced schema_id.
func buildSignedObject(event map[string]any) (map[string]any, error) {
	for _, f := range requiredFields {
		if v, ok := event[f]; !ok || v == nil {
			return nil, fmt.Errorf("missing required field: %s", f)
		}
	}
	out := map[string]any{}
	for _, f := range signedFields {
		v, ok := event[f]
		if !ok || v == nil {
			continue
		}
		if f == "occurred_at" || f == "ingested_at" {
			s, isStr := v.(string)
			if !isStr {
				return nil, fmt.Errorf("%s must be a string timestamp", f)
			}
			norm, err := NormalizeTS(s)
			if err != nil {
				return nil, err
			}
			out[f] = norm
		} else {
			out[f] = v
		}
	}
	out["schema_id"] = AuditSchemaID
	return out, nil
}

// stripNulls recursively removes null map members.
func stripNulls(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if val == nil {
				continue
			}
			out[k] = stripNulls(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, val := range x {
			out = append(out, stripNulls(val))
		}
		return out
	default:
		return v
	}
}

// CanonicalAuditBytes returns the canonical signed bytes for an audit event:
// build the signed object, strip nulls recursively, sort every object's keys
// deeply (alphabetical), emit compact UTF-8 JSON.
//
// encoding/json marshals map[string]any with keys sorted alphabetically and
// produces compact output, so a plain Marshal of the null-stripped signed
// object yields the canonical form.
func CanonicalAuditBytes(event map[string]any) ([]byte, error) {
	signed, err := buildSignedObject(event)
	if err != nil {
		return nil, err
	}
	stripped := stripNulls(signed)
	return json.Marshal(stripped)
}

// PayloadHashHex returns the lowercase hex SHA-256 of the canonical bytes.
func PayloadHashHex(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}
