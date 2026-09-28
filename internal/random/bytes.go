package random

import (
	"crypto/rand"
	"fmt"

	"github.com/arlogy/deltapilot/internal/failures"
)

// NewSecureBytes generates cryptographically secure random bytes.
func NewSecureBytes(length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause:   WrapSecureBytesGenerationError(),
			InternalCause: err,
		}
	}
	return buf, nil
}

func WrapSecureBytesGenerationError() error {
	return fmt.Errorf("%w: failed to generate secure random bytes", ErrDataGeneration)
}
