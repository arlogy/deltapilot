package random

import (
	"errors"
	"fmt"

	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/google/uuid"
)

var (
	ErrIDGeneration = errors.New("ID generation error")
)

func NewUUIDv7() (string, error) {
	newID, err := uuid.NewV7()
	if err != nil {
		return "", &failures.DetailedError{
			PublicCause:   WrapUUIDv7GenerationError(),
			InternalCause: err,
		}
	}

	return newID.String(), nil
}

func WrapUUIDv7GenerationError() error {
	return fmt.Errorf("%w: failed to generate UUIDv7", ErrIDGeneration)
}
