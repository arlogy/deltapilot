package testutils

import (
	"errors"
	"reflect"
	"testing"

	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/internal/failures"
)

type snapshotDeletionStore interface {
	DeleteByID(id string) (int64, error)
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

func CleanupStore(t *testing.T, store snapshotDeletionStore, snapshotIDs []string) {
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
