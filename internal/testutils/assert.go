package testutils

import (
	"errors"
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/google/go-cmp/cmp"
)

type AssertionResult struct {
	t          *testing.T
	successful bool
}

func (r AssertionResult) IsSuccessful() bool {
	return r.successful
}

func (r AssertionResult) Critical() {
	r.t.Helper()

	if !r.successful {
		r.t.FailNow()
		//r.t.Fatal("assertion failed to match expectation") // verbose alternative, thus not used
	}
}

func assertionPassed(t *testing.T) AssertionResult {
	return AssertionResult{t: t, successful: true}
}

func assertionFailed(t *testing.T) AssertionResult {
	return AssertionResult{t: t, successful: false}
}

func AssertEqual(t *testing.T, got any, want any, opts ...cmp.Option) AssertionResult {
	t.Helper()

	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertErrorIs(t *testing.T, got error, want error) AssertionResult {
	t.Helper()

	if !errors.Is(got, want) {
		t.Errorf("got [%v], want [%v]%s", got, want, func() string {
			if semanticErr := failures.AsErrorWithSemantics(got); semanticErr != got {
				return fmt.Sprintf("\n\ngot's semantics -> %v\n\n", semanticErr)
			}
			return ""
		}())
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertFieldNamesEqual(t *testing.T, data any, wantNames []string) AssertionResult {
	t.Helper()

	fieldsInfo := RequireFieldsMetadata(t, data, false)

	gotNames := make([]string, len(fieldsInfo))
	for i, f := range fieldsInfo {
		gotNames[i] = f.Declaration.Name
	}

	return AssertEqual(t, gotNames, wantNames)
}

func AssertNotEqual(t *testing.T, got any, other any, opts ...cmp.Option) AssertionResult {
	t.Helper()

	if cmp.Equal(got, other, opts...) {
		t.Errorf("want values to differ, got the same value:\n%#v", got)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertScalarPointersIndependent[T ~string](t *testing.T, p1 *T, p2 *T) AssertionResult {
	t.Helper()

	if p1 == p2 && p1 != nil {
		t.Errorf("got identical scalar pointers (%p), want distinct ones", p1)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertSlicesIndependent[T any](t *testing.T, s1 []T, s2 []T) AssertionResult {
	t.Helper()

	p1 := unsafe.SliceData(s1)
	p2 := unsafe.SliceData(s2)
	if p1 == p2 && p1 != nil {
		t.Errorf("got slices with identical data pointers (%p), want distinct ones", p1)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertTimeAfter(t *testing.T, got time.Time, other time.Time) AssertionResult {
	t.Helper()

	if !got.After(other) {
		t.Errorf("got [%v], want it to be after [%v]", got, other)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}

func AssertTimeLocationIs(t *testing.T, got time.Time, wantLoc *time.Location) AssertionResult {
	t.Helper()

	gotLoc := got.Location()
	if gotLoc != wantLoc {
		t.Errorf("got time location [%v], want [%v]; location pointers differ", gotLoc, wantLoc)
		return assertionFailed(t)
	}

	return assertionPassed(t)
}
