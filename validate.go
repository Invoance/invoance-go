package invoance

import (
	"fmt"
	"regexp"
)

var hexSha256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// assertSha256Hex validates that value is a 64-char lowercase hex SHA-256
// digest, returning a *Error of kind Validation otherwise.
func assertSha256Hex(fieldName, value string) error {
	if len(value) != 64 {
		return &Error{
			Kind:    KindValidation,
			Message: fmt.Sprintf("%s must be 64 hex chars (got %d chars)", fieldName, len(value)),
		}
	}
	if !hexSha256.MatchString(value) {
		prefix := value
		if len(prefix) > 16 {
			prefix = prefix[:16]
		}
		return &Error{
			Kind:    KindValidation,
			Message: fmt.Sprintf("%s must be lowercase hex [0-9a-f]; %q… is not", fieldName, prefix),
		}
	}
	return nil
}
