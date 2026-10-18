package random

import (
	"crypto/rand"

	"github.com/arlogy/deltapilot/internal/failures"
)

// NewSecureBytes generates cryptographically secure random bytes.
func NewSecureBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, WrapDataGenerationError(
			"failed to generate secure random bytes: length must be non-negative",
		)
	}

	buf := make([]byte, n)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause:   WrapDataGenerationError("failed to generate secure random bytes"),
			InternalCause: err,
		}
	}

	return buf, nil
}
