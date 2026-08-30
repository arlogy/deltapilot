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

func TestChronicleStoreCanShareState(t *testing.T) {
	testutils.TestChronicleStoreCanShareState(t, makeChronicleStore, true)
}

func TestNewChronicleStore(t *testing.T) {
	t.Run("creates an empty snapshot store", func(t *testing.T) {
		handle := testutils.InitDB(t)

		testutils.CheckStoreEmpty[revision.ChronicleSnapshot](t, handle)

		sqldb.NewChronicleStore(handle)
		testutils.CheckStoreEmpty[revision.ChronicleSnapshot](t, handle)

		// note: this test also helps ensure that all snapshots created during a previous run are deleted
		//       during that run
	})
}

func TestChronicleStoreAddSnapshot(t *testing.T) {
	testutils.TestChronicleStoreAddSnapshot(
		t,
		makeChronicleStore,
		func(snapshot *revision.ChronicleSnapshot) (error, error) {
			return storage.ErrSnapshotDuplicate, &failures.DetailedError{
				PublicCause: storage.WrapSnapshotDuplicateUncategorized(
					snapshot.ID,
					snapshot.ScopeID,
					snapshot.ResourceID,
					snapshot.VariantID,
				),
				//InternalCause: errors.New("..."), // not used in tests since its value is dialect-dependent
			}
		},
		time.Microsecond,
		false,
	)
}

func TestChronicleStoreGetByID(t *testing.T) {
	testutils.TestChronicleStoreGetByID(t, makeChronicleStore, time.Microsecond, false)
}

func TestChronicleStoreGetByScopeID(t *testing.T) {
	testutils.TestChronicleStoreGetByScopeID(t, makeChronicleStore, time.Microsecond, false)
}

func TestChronicleStoreGetByScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestChronicleStoreGetByScopeAndResourceAndVariant(
		t, makeChronicleStore, time.Microsecond, false,
	)
}

func TestChronicleStoreMapsID(t *testing.T) {
	testutils.TestChronicleStoreMapsID(t, makeChronicleStore, time.Microsecond)
}

func TestChronicleStoreMapsScopeID(t *testing.T) {
	testutils.TestChronicleStoreMapsScopeID(t, makeChronicleStore, time.Microsecond)
}

func TestChronicleStoreMapsScopeAndResourceAndVariant(t *testing.T) {
	testutils.TestChronicleStoreMapsScopeAndResourceAndVariant(t, makeChronicleStore, time.Microsecond)
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
	handle := testutils.InitDB(t)
	return sqldb.NewChronicleStore(handle)
}
