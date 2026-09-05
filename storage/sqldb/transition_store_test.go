package sqldb_test

import (
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
	"github.com/arlogy/deltapilot/storage/sqldb"
)

func TestTransitionStoreCanShareState(t *testing.T) {
	testutils.TestTransitionStoreCanShareState(t, makeTransitionStore, true)
}

func TestNewTransitionStore(t *testing.T) {
	t.Run("creates an empty snapshot store", func(t *testing.T) {
		handle := testutils.InitDB(t)

		testutils.CheckStoreEmpty[revision.TransitionSnapshot](t, handle)

		sqldb.NewTransitionStore(handle)
		testutils.CheckStoreEmpty[revision.TransitionSnapshot](t, handle)

		// note: this test also helps ensure that all snapshots created during a previous run are deleted
		//       during that run
	})
}

func TestTransitionStoreAddSnapshot(t *testing.T) {
	testutils.TestTransitionStoreAddSnapshot(
		t,
		makeTransitionStore,
		func(snapshot *revision.TransitionSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, &failures.DetailedError{
				PublicCause: storage.WrapSnapshotDuplicateUncategorized(
					snapshot.ID,
					&snapshot.ScopeID,
					snapshot.ResourceID,
					&snapshot.VariantID,
				),
				//InternalCause: errors.New("..."), // not used in tests since its value is dialect-dependent
			}
		},
		func(snapshot *revision.TransitionSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, &failures.DetailedError{
				PublicCause: storage.WrapSnapshotDuplicateUncategorized(
					snapshot.ID,
					&snapshot.ScopeID,
					snapshot.ResourceID,
					&snapshot.VariantID,
				),
				//InternalCause: errors.New("..."), // not used in tests since its value is dialect-dependent
			}
		},
		time.Microsecond,
		false,
	)
}

func TestTransitionStoreGetByID(t *testing.T) {
	testutils.TestTransitionStoreGetByID(t, makeTransitionStore, time.Microsecond, false)
}

func TestTransitionStoreGetByScopeID(t *testing.T) {
	testutils.TestTransitionStoreGetByScopeID(t, makeTransitionStore, time.Microsecond, false)
}

func TestTransitionStoreGetByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionStoreGetByScopeAndResourceAndVariant(
		t, makeTransitionStore, time.Microsecond, false,
	)
}

func TestTransitionStoreMapsID(t *testing.T) {
	testutils.TestTransitionStoreMapsID(t, makeTransitionStore, time.Microsecond)
}

func TestTransitionStoreMapsScopeID(t *testing.T) {
	testutils.TestTransitionStoreMapsScopeID(t, makeTransitionStore, time.Microsecond)
}

func TestTransitionStoreMapsScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionStoreMapsScopeAndResourceAndVariant(t, makeTransitionStore, time.Microsecond)
}

func TestTransitionApplyTransitionForID(t *testing.T) {
	testutils.TestTransitionApplyTransitionForID(t, makeTransitionStore, time.Microsecond, false)
}

func TestTransitionApplyTransitionForScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionApplyTransitionForScopeAndResourceAndVariant(
		t, makeTransitionStore, time.Microsecond, false,
	)
}

func TestTransitionStoreDeleteByID(t *testing.T) {
	testutils.TestTransitionStoreDeleteByID(t, makeTransitionStore)
}

func TestTransitionStoreDeleteByScopeID(t *testing.T) {
	testutils.TestTransitionStoreDeleteByScopeID(t, makeTransitionStore)
}

func TestTransitionStoreDeleteByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionStoreDeleteByScopeAndResourceAndVariant(t, makeTransitionStore)
}

func makeTransitionStore(t *testing.T) storage.TransitionStore {
	handle := testutils.InitDB(t)
	return sqldb.NewTransitionStore(handle)
}
