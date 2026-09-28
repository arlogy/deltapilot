package testutils

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/arlogy/deltapilot/internal/random"
)

func GenerateAlphanumeric(t *testing.T) string {
	t.Helper()
	return random.NewAlphanumeric(random.RandomSequenceLen)
}

func GenerateBytes(t *testing.T) []byte {
	t.Helper()

	data, err := random.NewSecureBytes(random.RandomSequenceLen)
	if err != nil {
		t.Fatalf("%v", failures.AsErrorWithSemantics(err))
	}

	return data
}

func GenerateID(t *testing.T) string {
	t.Helper()

	id, err := random.NewUUIDv7()
	if err != nil {
		t.Fatalf("%v", failures.AsErrorWithSemantics(err))
	}

	return id
}

func GeneratePointerID(t *testing.T) *string {
	t.Helper()
	return new(GenerateID(t))
}

func GenerateTimeZone() *time.Location {
	// note: rand.Int() alone would have produced arbitrary offsets
	offset := rand.IntN(26*60*60+1) - 12*60*60 // realistic UTC offset; range [-12h, +14h]
	return time.FixedZone("Test", offset)
}
