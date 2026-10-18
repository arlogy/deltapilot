package random

import (
	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/google/uuid"
)

// NewUUIDv7 generates a UUIDv7 string.
func NewUUIDv7() (string, error) {
	newID, err := uuid.NewV7()
	if err != nil {
		return "", &failures.DetailedError{
			PublicCause:   WrapDataGenerationError("failed to generate UUIDv7"),
			InternalCause: err,
		}
	}

	return newID.String(), nil
}
