package random

import "errors"

const RandomSequenceLen = 20 // large enough to make collisions unlikely for random byte sequence generation

var (
	ErrDataGeneration = errors.New("data generation error")
)
