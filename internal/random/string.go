package random

import "math/rand/v2"

const asciiChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const asciiLen = len(asciiChars)

// NewAlphanumeric generates a random string containing only ASCII letters and digits.
func NewAlphanumeric(n int) (string, error) {
	if n < 0 {
		return "", WrapDataGenerationError(
			"failed to generate alphanumeric string: length must be non-negative",
		)
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = asciiChars[rand.IntN(asciiLen)]
	}

	return string(b), nil
}
