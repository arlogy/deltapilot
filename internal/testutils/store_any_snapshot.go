package testutils

import (
	"errors"
	"reflect"
	"testing"

	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/internal/failures"
)

type storeAddDelete[T any] interface {
	AddSnapshot(snapshot *T) error
	storeDeleteOnly
}

type storeDeleteOnly interface {
	DeleteByID(id string) (int64, error)
}

type storeFixtureData[T any] struct {
	pAttrsID1 string // ID generated for a snapshot that preserves all supplied attributes
	pAttrsID2 string // ID generated for a snapshot that preserves all supplied attributes

	withStore0 func(fn func(store T))
	withStore1 func(fn func(store T))
	withStoreN func(fn func(store T))
}

func CheckStoreEmpty[T any](t *testing.T, handle *dbclient.DBHandle) {
	t.Helper()

	var snapshot T
	var count int64

	err := handle.DB.Model(&snapshot).Count(&count).Error
	if err != nil {
		t.Fatalf("failed to check that store is empty: %v", err)
	}
	if count > 0 {
		t.Fatalf("failed to check that store is empty: number of snapshots is %d", count)
	}
}

func CheckStoreOutputError(t *testing.T, gotErr error, wrappedErr error, returnedErr error) {
	t.Helper()

	AssertErrorIs(t, gotErr, wrappedErr).Critical()
	AssertErrorIs(t, returnedErr, wrappedErr).Critical()

	AssertEqual(t, reflect.TypeOf(gotErr) == reflect.TypeOf(returnedErr), true).Critical()
	AssertEqual(t, gotErr.Error(), returnedErr.Error()).Critical()

	if gotErr, ok := errors.AsType[*failures.DetailedError](gotErr); ok {
		returnedErr, ok := errors.AsType[*failures.DetailedError](returnedErr)
		AssertEqual(t, ok, true).Critical()
		if ok {
			AssertEqual(t, gotErr.PublicCause.Error(), returnedErr.PublicCause.Error()).Critical()
			AssertEqual(t, len(gotErr.InternalCause.Error()) > 0, true).Critical()

			// detect newly added fields so they can be accounted for in tests when necessary
			AssertFieldNamesEqual(t, gotErr, []string{"PublicCause", "InternalCause"}).Critical()
		}
	}
}

func CleanupStore(t *testing.T, store storeDeleteOnly, snapshotIDs []string) {
	for i, id := range snapshotIDs {
		_, err := store.DeleteByID(id)
		if err != nil {
			t.Fatalf(
				"failed to delete snapshot for ID %q at index %d: %v",
				id,
				i,
				failures.AsErrorWithSemantics(err),
			)
		}
		// failing when the number of deleted snapshots is not 1 is irrelevant because the snapshot may have
		// already been deleted or may have failed to be added to the store; the caller is therefore expected
		// to call CleanupStore() with snapshot IDs that may exist in the store, since snapshots that were
		// never added to the store will be ignored
		//t.Fatalf("expected 1 snapshot deletion; deleted %d", count)
	}
}

// NewStoreFixture creates a store fixture.
//
// Note that TStore is assumed to be substituted with an interface type rather than a struct type, so we
// always use TStore instead of *TStore for both function parameters and return types.
func NewStoreFixture[TSnapshot any, TStore storeAddDelete[TSnapshot]](
	t *testing.T,
	newStore func(t *testing.T) TStore,
	newSnapshot func(t *testing.T, preserveAttributes bool) (string, *TSnapshot),
) storeFixtureData[TStore] {
	preservedAttrsID1, snapshot1 := newSnapshot(t, true)
	preservedAttrsID2, snapshot2 := newSnapshot(t, true)
	generatedAttrsID_, snapshot3 := newSnapshot(t, false)

	return storeFixtureData[TStore]{
		pAttrsID1: preservedAttrsID1,
		pAttrsID2: preservedAttrsID2,

		// store0, store1 and storeN are created within their respective callbacks and cleaned up when the
		// callbacks return, so that the stores remain independent and do not affect one another. Indeed, as
		// implied by storage.ChronicleStore.CanShareState(), different store instances may share the same
		// underlying state when tested against a single storage backend (e.g. a database).

		withStore0: func(fn func(store TStore)) {
			store0 := newStore(t) // empty store
			fn(store0)
		},

		withStore1: func(fn func(store TStore)) {
			store1 := newStore(t) // store with one snapshot
			for i, snapshot := range []*TSnapshot{snapshot1} {
				err := store1.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to store1: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, store1, []string{preservedAttrsID1})
			fn(store1)
		},

		withStoreN: func(fn func(store TStore)) {
			storeN := newStore(t) // store with several snapshots
			for i, snapshot := range []*TSnapshot{snapshot1, snapshot2, snapshot3} {
				err := storeN.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to storeN: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, storeN, []string{preservedAttrsID1, preservedAttrsID2, generatedAttrsID_})
			fn(storeN)
		},
	}
}
