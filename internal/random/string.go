package random

import "math/rand/v2"

const asciiChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const asciiLen = len(asciiChars)

// NewAlphanumeric generates a random string containing only ASCII letters and digits.
func NewAlphanumeric(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = asciiChars[rand.IntN(asciiLen)]
	}
	return string(b)
}
