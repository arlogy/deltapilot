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

func TestChronicleStoreCanShareState(t *testing.T) {
	testutils.TestChronicleStoreCanShareState(t, makeChronicleStore, false)
}

func TestNewChronicleStore(t *testing.T) {
	t.Run("creates an empty snapshot store", func(t *testing.T) {
		store := memdb.NewChronicleStore()

		fieldsInfo := testutils.RequireFieldsMetadata(t, store, true)

		testutils.AssertEqual(t, len(fieldsInfo), 2)

		testutils.AssertEqual(t, fieldsInfo[0].Declaration.Name, "mu")
		testutils.AssertEqual(t, fieldsInfo[0].Declaration.Type == reflect.TypeOf(sync.RWMutex{}), true)

		testutils.AssertEqual(t, fieldsInfo[1].Declaration.Name, "snapshots")
		testutils.AssertEqual(t, fieldsInfo[1].Initialization.Type().Kind(), reflect.Map)
		testutils.AssertEqual(t, fieldsInfo[1].Initialization.Len(), 0)
	})
}

func TestChronicleStoreAddSnapshot(t *testing.T) {
	testutils.TestChronicleStoreAddSnapshot(
		t,
		makeChronicleStore,
		func(snapshot *revision.ChronicleSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, storage.WrapSnapshotDuplicateID(snapshot.ID)
		},
		0,
		true,
	)
}

func TestChronicleStoreGetByID(t *testing.T) {
	testutils.TestChronicleStoreGetByID(t, makeChronicleStore, 0, true)
}

func TestChronicleStoreGetByScopeID(t *testing.T) {
	testutils.TestChronicleStoreGetByScopeID(t, makeChronicleStore, 0, true)
}

func TestChronicleStoreGetByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestChronicleStoreGetByScopeAndResourceAndVariant(t, makeChronicleStore, 0, true)
}

func TestChronicleStoreMapsID(t *testing.T) {
	testutils.TestChronicleStoreMapsID(t, makeChronicleStore, 0)
}

func TestChronicleStoreMapsScopeID(t *testing.T) {
	testutils.TestChronicleStoreMapsScopeID(t, makeChronicleStore, 0)
}

func TestChronicleStoreMapsScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestChronicleStoreMapsScopeAndResourceAndVariant(t, makeChronicleStore, 0)
}

func TestChronicleStoreDeleteByID(t *testing.T) {
	testutils.TestChronicleStoreDeleteByID(t, makeChronicleStore)
}

func TestChronicleStoreDeleteByScopeID(t *testing.T) {
	testutils.TestChronicleStoreDeleteByScopeID(t, makeChronicleStore)
}

func TestChronicleStoreDeleteByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestChronicleStoreDeleteByScopeAndResourceAndVariant(t, makeChronicleStore)
}

func makeChronicleStore(t *testing.T) storage.ChronicleStore {
	return memdb.NewChronicleStore()
}
