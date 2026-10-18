package random

import (
	"errors"
	"fmt"
)

const RandomSequenceLen = 20 // large enough to make collisions unlikely for random byte sequence generation

var (
	ErrDataGeneration = errors.New("data generation error")
)

func WrapDataGenerationError(reason string) error {
	return fmt.Errorf("%w: %s", ErrDataGeneration, reason)
}
