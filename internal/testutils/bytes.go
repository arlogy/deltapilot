package testutils

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

func PlaceByteRandomly(t *testing.T, data []byte, b byte, count int) {
	t.Helper()

	n := len(data)

	if count < 0 || count > n {
		t.Fatal("failed to place byte randomly: repetition count is out of range")
	}

	for _, i := range rand.Perm(n)[:count] {
		data[i] = b
	}
}

func PrefixBeforeByte(data []byte, b byte) []byte {
	result := data
	if i := bytes.IndexByte(result, b); i >= 0 {
		result = result[:i]
	}

	return bytes.Clone(result)
}
