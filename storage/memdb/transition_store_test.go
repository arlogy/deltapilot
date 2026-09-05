package memdb_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
	"github.com/arlogy/deltapilot/storage/memdb"
)

func TestTransitionStoreCanShareState(t *testing.T) {
	testutils.TestTransitionStoreCanShareState(t, makeTransitionStore, false)
}

func TestNewTransitionStore(t *testing.T) {
	t.Run("creates an empty snapshot store", func(t *testing.T) {
		store := memdb.NewTransitionStore()

		fieldsInfo := testutils.RequireFieldsMetadata(t, store, true)

		testutils.AssertEqual(t, len(fieldsInfo), 2)

		testutils.AssertEqual(t, fieldsInfo[0].Declaration.Name, "mu")
		testutils.AssertEqual(t, fieldsInfo[0].Declaration.Type == reflect.TypeOf(sync.RWMutex{}), true)

		testutils.AssertEqual(t, fieldsInfo[1].Declaration.Name, "snapshots")
		testutils.AssertEqual(t, fieldsInfo[1].Initialization.Type().Kind(), reflect.Map)
		testutils.AssertEqual(t, fieldsInfo[1].Initialization.Len(), 0)
	})
}

func TestTransitionStoreAddSnapshot(t *testing.T) {
	testutils.TestTransitionStoreAddSnapshot(
		t,
		makeTransitionStore,
		func(snapshot *revision.TransitionSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, storage.WrapSnapshotDuplicateID(snapshot.ID)
		},
		func(snapshot *revision.TransitionSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, storage.WrapSnapshotDuplicateScopeAndResourceAndVariant(
				snapshot.ScopeID, snapshot.ResourceID, snapshot.VariantID,
			)
		},
		0,
		true,
	)
}

func TestTransitionStoreGetByID(t *testing.T) {
	testutils.TestTransitionStoreGetByID(t, makeTransitionStore, 0, true)
}

func TestTransitionStoreGetByScopeID(t *testing.T) {
	testutils.TestTransitionStoreGetByScopeID(t, makeTransitionStore, 0, true)
}

func TestTransitionStoreGetByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionStoreGetByScopeAndResourceAndVariant(t, makeTransitionStore, 0, true)
}

func TestTransitionStoreMapsID(t *testing.T) {
	testutils.TestTransitionStoreMapsID(t, makeTransitionStore, 0)
}

func TestTransitionStoreMapsScopeID(t *testing.T) {
	testutils.TestTransitionStoreMapsScopeID(t, makeTransitionStore, 0)
}

func TestTransitionStoreMapsScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionStoreMapsScopeAndResourceAndVariant(t, makeTransitionStore, 0)
}

func TestTransitionApplyTransitionForID(t *testing.T) {
	testutils.TestTransitionApplyTransitionForID(t, makeTransitionStore, 0, true)
}

func TestTransitionApplyTransitionForScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestTransitionApplyTransitionForScopeAndResourceAndVariant(t, makeTransitionStore, 0, true)
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
	return memdb.NewTransitionStore()
}
