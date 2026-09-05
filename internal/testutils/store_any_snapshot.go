package testutils

import (
	"errors"
	"reflect"
	"testing"

	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/arlogy/deltapilot/storage"
)

// Note: TStore is assumed to be substituted with an interface type rather than a struct type, so we always
//       use TStore instead of *TStore for both function parameters and return types.

type storeFixtureData[TStore any] struct {
	recordedIDs []string // IDs of snapshots added to the fixture stores

	withStore0 func(fn func(store TStore))
	withStore1 func(fn func(store TStore))
	withStoreN func(fn func(store TStore))
}

type storeOpAdd[TSnapshot any] interface {
	AddSnapshot(snapshot *TSnapshot) error
}

type storeOpGetByID[TSnapshot any] interface {
	GetByID(id string) (*TSnapshot, error)
}

type storeOpDelete interface {
	DeleteByID(id string) (int64, error)
}

func AddSnapshotsToStore[TSnapshot any](t *testing.T, store storeOpAdd[TSnapshot], snapshots []*TSnapshot) {
	t.Helper()

	for i, snapshot := range snapshots {
		err := store.AddSnapshot(snapshot)
		if err != nil {
			t.Fatalf("failed to add snapshot at index %d to store: %v", i, failures.AsErrorWithSemantics(err))
		}
	}
}

func CheckStoreEmpty[TSnapshot any](t *testing.T, handle *dbclient.DBHandle) {
	t.Helper()

	var snapshot TSnapshot
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

func CheckStoreStateSharing[
	TStore interface {
		storeOpAdd[TSnapshot]
		storeOpDelete
		storeOpGetByID[TSnapshot]
	},
	TSnapshot any,
](
	t *testing.T,
	newStore func(t *testing.T) TStore,
	newSnapshot func(t *testing.T) (string, *TSnapshot),
	stateSharable bool,
) {
	t.Helper()

	emptyStore := newStore(t)

	id, baseSnapshot := newSnapshot(t)

	gotSnapshot, gotErr := emptyStore.GetByID(id)
	AssertEqual(t, gotSnapshot == nil, true).Critical()
	AssertErrorIs(t, gotErr, storage.ErrSnapshotRetrieval).Critical()
	AssertEqual(t, gotErr.Error(), storage.WrapSnapshotNotFoundByID(id).Error()).Critical()

	seededStore := newStore(t)
	errAdd := seededStore.AddSnapshot(baseSnapshot)
	defer CleanupStore(t, seededStore, []string{id})

	AssertErrorIs(t, errAdd, nil).Critical()

	gotSnapshot, gotErr = emptyStore.GetByID(id)
	if stateSharable {
		// note: because tests run against a single storage backend (e.g. a database), stateSharable being
		//       true is equivalent to the state being shared among stores
		AssertEqual(t, gotSnapshot != nil, true).Critical()
		AssertErrorIs(t, gotErr, nil).Critical()
	} else {
		AssertEqual(t, gotSnapshot == nil, true).Critical()
		AssertErrorIs(t, gotErr, storage.ErrSnapshotRetrieval).Critical()
		AssertEqual(t, gotErr.Error(), storage.WrapSnapshotNotFoundByID(id).Error()).Critical()
	}
}

func CleanupStore(t *testing.T, store storeOpDelete, snapshotIDs []string) {
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

func GenerateSnapshots[TSnapshot any](
	t *testing.T,
	newSnapshot1 func(t *testing.T) *TSnapshot,
	newSnapshot2 func(t *testing.T) *TSnapshot,
	includeNil bool,
	count int,
) []*TSnapshot {
	t.Helper()

	snapshots := make([]*TSnapshot, count)
	if includeNil {
		for i := range count {
			// arbitrary variation pattern; callers should rely on generated contents only
			switch i % 3 {
			case 0:
				snapshots[i] = nil
			case 1:
				snapshots[i] = newSnapshot1(t)
			default:
				snapshots[i] = newSnapshot2(t)
			}
		}
	} else {
		for i := range count {
			// arbitrary variation pattern; callers should rely on generated contents only
			switch i % 2 {
			case 0:
				snapshots[i] = newSnapshot1(t)
			default:
				snapshots[i] = newSnapshot2(t)
			}
		}
	}

	return snapshots
}

func NewStoreFixture[
	TStore interface {
		storeOpAdd[TSnapshot]
		storeOpDelete
	},
	TSnapshot any,
](
	t *testing.T,
	newStore func(t *testing.T) TStore,
	newSnapshots func(t *testing.T) ([]string, []*TSnapshot),
) storeFixtureData[TStore] {
	t.Helper()

	ids, snapshots := newSnapshots(t)

	AssertEqual(t, len(snapshots) >= 3, true).Critical()
	AssertEqual(t, len(snapshots), len(ids)).Critical()

	return storeFixtureData[TStore]{
		recordedIDs: ids,

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
			for i, snapshot := range []*TSnapshot{snapshots[0]} {
				err := store1.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to store1: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, store1, []string{ids[0]})
			fn(store1)
		},

		withStoreN: func(fn func(store TStore)) {
			storeN := newStore(t) // store with several snapshots
			for i, snapshot := range snapshots {
				err := storeN.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to storeN: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, storeN, ids)
			fn(storeN)
		},
	}
}
