package random

import "math/rand/v2"

const asciiChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const asciiLen = len(asciiChars)

const UniqueStrLen = 20 // long enough for random string generation while avoiding duplicates

func NewAlphanumeric(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = asciiChars[rand.IntN(asciiLen)]
	}
	return string(b)
}
