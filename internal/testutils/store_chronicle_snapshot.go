package testutils

import (
	stdcmp "cmp"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/arlogy/deltapilot/internal/ptr"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Notes on this file's test flow.
// - Some assertions establish prerequisites for other assertions.
// - Some flows are repeated ScenarioRepeatCount times for consistency.
// - We therefore use Critical() to prevent cascading errors when a prerequisite fails.

func GenerateChronicleSnapshotItem(t *testing.T) *revision.ChronicleSnapshot {
	t.Helper()

	id := GenerateID(t)
	scopeID := GeneratePointerID(t)
	resourceID := GenerateID(t)
	variantID := GeneratePointerID(t)
	data := GenerateBytes(t)

	// all nullable fields are set to non-null values for the snapshot
	return revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
}

func GenerateChronicleSnapshotSlice(
	t *testing.T,
	scopeID *string,
	variantID *string,
	data []byte,
	includeNil bool,
	count int,
) []*revision.ChronicleSnapshot {
	t.Helper()

	snapshots := make([]*revision.ChronicleSnapshot, count)
	if includeNil {
		for i := range count {
			// arbitrary variation pattern; callers should rely on generated contents only
			switch i % 3 {
			case 0:
				snapshots[i] = nil
			case 1:
				id := GenerateID(t)
				resourceID := GenerateID(t)
				snapshots[i] = revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
			default:
				snapshots[i] = GenerateChronicleSnapshotItem(t)
			}
		}
	} else {
		for i := range count {
			// arbitrary variation pattern; callers should rely on generated contents only
			switch i % 2 {
			case 0:
				id := GenerateID(t)
				resourceID := GenerateID(t)
				snapshots[i] = revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
			default:
				snapshots[i] = GenerateChronicleSnapshotItem(t)
			}
		}
	}

	return snapshots
}

func GenerateChronicleSnapshotsInStore(
	t *testing.T,
	store storage.ChronicleStore,
	scopeID *string,
	variantID *string,
	data []byte,
	count int,
) []*revision.ChronicleSnapshot {
	t.Helper()

	snapshots := GenerateChronicleSnapshotSlice(t, scopeID, variantID, data, false, count)

	for i, snapshot := range snapshots {
		err := store.AddSnapshot(snapshot)
		if err != nil {
			t.Fatalf("failed to add snapshot at index %d to store: %v", i, failures.AsErrorWithSemantics(err))
		}
	}

	return snapshots
}

func TestChronicleStoreCanShareState(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	stateSharable bool,
) {
	t.Run("returns "+fmt.Sprintf("%t", stateSharable), func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			testStateSharing := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got := store.CanShareState()
					AssertEqual(t, got, stateSharable).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testStateSharing(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testStateSharing(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testStateSharing(store)
			})

			// test snapshot retrieval from a store where it was not added
			// note: because tests run against a single storage backend (e.g. a database), stateSharable being
			//       true is equivalent to the state being shared among stores
			func() {
				emptyStore := newStore(t)

				id := GenerateID(t)
				resourceID := GenerateID(t)
				snapshot := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)

				gotSnapshot, gotErr := emptyStore.GetByID(id)
				AssertEqual(t, gotSnapshot == nil, true).Critical()
				AssertErrorIs(t, gotErr, storage.ErrSnapshotRetrieval).Critical()
				AssertEqual(t, gotErr.Error(), fmt.Sprintf("snapshot cannot be retrieved: ID %q", id)).
					Critical()

				seededStore := newStore(t)
				errAdd := seededStore.AddSnapshot(snapshot)
				defer CleanupStore(t, seededStore, []string{id})

				AssertErrorIs(t, errAdd, nil).Critical()

				gotSnapshot, gotErr = emptyStore.GetByID(id)
				if stateSharable {
					AssertEqual(t, gotSnapshot != nil, true).Critical()
					AssertErrorIs(t, gotErr, nil).Critical()
				} else {
					AssertEqual(t, gotSnapshot == nil, true).Critical()
					AssertErrorIs(t, gotErr, storage.ErrSnapshotRetrieval).Critical()
					AssertEqual(t, gotErr.Error(), fmt.Sprintf("snapshot cannot be retrieved: ID %q", id)).
						Critical()
				}
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreAddSnapshot(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	duplicateIDsErrors func(snapshot *revision.ChronicleSnapshot) (error, error),
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects a nil snapshot", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			testAddNilSnapshot := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					err := store.AddSnapshot(nil)
					AssertErrorIs(t, err, storage.ErrSnapshotRequired).Critical()
					AssertEqual(t, err.Error(), "snapshot is required").Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testAddNilSnapshot(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testAddNilSnapshot(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testAddNilSnapshot(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("rejects a snapshot with a duplicate ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			id := GenerateID(t)
			resourceID := GenerateID(t)
			defer CleanupStore(t, store, []string{id})

			// initial case
			func() {
				snapshot := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
				err1 := store.AddSnapshot(snapshot)

				AssertErrorIs(t, err1, nil).Critical()

				for range ScenarioRepeatCount {
					err2 := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateIDsErrors(snapshot)
					CheckStoreOutputError(t, err2, wrappedErr, returnedErr)
				}
			}()

			// another case, sharing identical snapshot contents
			func() {
				snapshot := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)

				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateIDsErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()

			// another case, sharing only the snapshot ID
			func() {
				snapshot := GenerateChronicleSnapshotItem(t)
				snapshot.ID = id

				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateIDsErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("stores an independent copy of the snapshot when accepted", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			for range ScenarioRepeatCount {
				id := GenerateID(t)
				resourceID := GenerateID(t)

				refSnapshot := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())

				addedSnapshot := refSnapshot.Clone()
				errAdd := store.AddSnapshot(addedSnapshot)
				defer CleanupStore(t, store, []string{id})

				gotSnapshot, errGet := store.GetByID(id)

				assertAddedChronicleSnapshotPreserved(
					t, errAdd, errGet, refSnapshot, addedSnapshot, gotSnapshot, timeOpts,
					preserveTimeLocation,
				)
				assertRetrievedChronicleSnapshotIndependent(t, addedSnapshot, gotSnapshot)

				// detect newly added fields so they can be accounted for in tests when necessary
				AssertFieldNamesEqual(t, gotSnapshot, []string{
					"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
				}).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotSlice(t, scopeID, variantID, data, true, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctAddCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					err := store.AddSnapshot(snapshots[workerIdx])
					if err == nil {
						atomic.AddInt64(&result, 1)
					} else if snapshots[workerIdx] == nil && errors.Is(err, storage.ErrSnapshotRequired) {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctAddCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				if snapshots[i] != nil {
					gotSnapshot, err := store.GetByID(snapshots[i].ID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, gotSnapshot == snapshots[i], false).Critical()
					AssertEqual(t, gotSnapshot, snapshots[i], timeOpts...).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreGetByID(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects an unknown snapshot ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			unknownID := GenerateID(t)

			testGetByUnknownID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.GetByID(unknownID)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertEqual(
						t, err.Error(), fmt.Sprintf("snapshot cannot be retrieved: ID %q", unknownID),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testGetByUnknownID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testGetByUnknownID(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testGetByUnknownID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("yields an independent snapshot copy for a known ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			id := GenerateID(t)
			resourceID := GenerateID(t)

			refSnapshot := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)
			refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())

			addedSnapshot := refSnapshot.Clone()
			errAdd := store.AddSnapshot(addedSnapshot)
			defer CleanupStore(t, store, []string{id})

			for range ScenarioRepeatCount {
				gotSnapshot, errGet := store.GetByID(id)

				assertAddedChronicleSnapshotPreserved(
					t, errAdd, errGet, refSnapshot, addedSnapshot, gotSnapshot, timeOpts,
					preserveTimeLocation,
				)
				assertRetrievedChronicleSnapshotIndependent(t, addedSnapshot, gotSnapshot)

				// detect newly added fields so they can be accounted for in tests when necessary
				AssertFieldNamesEqual(t, gotSnapshot, []string{
					"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
				}).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctGetCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					snapshot, err := store.GetByID(snapshots[workerIdx].ID)
					if err == nil && snapshot != nil {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctGetCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreGetByScopeID(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("yields an empty snapshot slice for an unknown scope ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			unknownScopeID := GeneratePointerID(t)

			testGetByUnknownScopeID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.GetByScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testGetByUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testGetByUnknownScopeID(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testGetByUnknownScopeID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	ordering := ", in unspecified order"
	t.Run("yields independent snapshot copies for a known scope ID"+ordering, func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			id1 := GenerateID(t)
			id2 := GenerateID(t)
			resourceID := GenerateID(t)

			refSnapshots := []*revision.ChronicleSnapshot{
				// snapshots that store.GetByScopeID(scopeID) will retrieve
				revision.NewChronicleSnapshot(id1, scopeID, resourceID, variantID, data),
				revision.NewChronicleSnapshot(id2, scopeID, resourceID, variantID, data),

				// snapshots that store.GetByScopeID(scopeID) will fail to retrieve
				// note: they are set later so their IDs sort last when sortSnapshots() is called
				nil,

				// snapshots that store.GetByScopeID(scopeID) will retrieve
				func() *revision.ChronicleSnapshot {
					result := GenerateChronicleSnapshotItem(t)
					result.ScopeID = scopeID
					return result
				}(),
			}
			refSnapshots[2] = GenerateChronicleSnapshotItem(t)
			for _, refSnapshot := range refSnapshots {
				AssertEqual(t, refSnapshot == nil, false).Critical() // make sure each nil slot was filled
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
			}

			addedSnapshots := make([]*revision.ChronicleSnapshot, len(refSnapshots))
			errAdds := make([]error, len(refSnapshots))
			for i := range refSnapshots {
				addedSnapshots[i] = refSnapshots[i].Clone()
				errAdds[i] = store.AddSnapshot(addedSnapshots[i])
			}
			defer CleanupStore(t, store, chronicleSnapshots2IDs(addedSnapshots))

			// store.GetByScopeID() does not guarantee any particular ordering; so we sort before comparison
			sortSnapshots := func(snapshots []*revision.ChronicleSnapshot) []*revision.ChronicleSnapshot {
				snapshots = slices.Clone(snapshots)
				slices.SortFunc(snapshots, func(a, b *revision.ChronicleSnapshot) int {
					return stdcmp.Compare(a.ID, b.ID)
				})
				return snapshots
			}

			refSorted := sortSnapshots(refSnapshots)
			addedSorted := sortSnapshots(addedSnapshots)
			for _, errAdd := range errAdds {
				AssertErrorIs(t, errAdd, nil).Critical()
			}

			for range ScenarioRepeatCount {
				gotSnapshots, errGet := store.GetByScopeID(scopeID)
				gotSorted := sortSnapshots(gotSnapshots)

				AssertErrorIs(t, errGet, nil).Critical()
				AssertEqual(t, gotSorted, refSorted[:len(refSorted)-1], timeOpts...).Critical()
				AssertEqual(t, gotSorted, addedSorted[:len(addedSorted)-1], timeOpts...).Critical()

				for i, gotSnapshot := range gotSorted {
					refSnapshot, addedSnapshot := refSorted[i], addedSorted[i]

					assertAddedChronicleSnapshotPreserved(
						t, nil, nil, refSnapshot, addedSnapshot, gotSnapshot, timeOpts, preserveTimeLocation,
					)
					assertRetrievedChronicleSnapshotIndependent(t, addedSnapshot, gotSnapshot)

					// detect newly added fields so they can be accounted for in tests when necessary
					AssertFieldNamesEqual(t, gotSnapshot, []string{
						"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
					}).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			matchCount := 0
			for _, snapshot := range snapshots {
				if ptr.EqualString(snapshot.ScopeID, scopeID) {
					matchCount++
				}
			}

			correctGetCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					snapshots, err := store.GetByScopeID(snapshots[workerIdx].ScopeID)
					if err == nil && len(snapshots) == matchCount {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctGetCount, int64(matchCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreGetByScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("yields an empty snapshot slice for an unknown scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			resourceID := fixture.resourceID

			unknownScopeID := GeneratePointerID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GeneratePointerID(t)

			testGetByUnknownCombination := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					// 3 fields unknown

					got, err := store.GetByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					// 2 fields unknown

					got, err = store.GetByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, variantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(
						unknownScopeID, resourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(
						scopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					// 1 field unknown

					got, err = store.GetByScopeAndResourceAndVariant(unknownScopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(scopeID, unknownResourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(scopeID, resourceID, unknownVariantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testGetByUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testGetByUnknownCombination(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testGetByUnknownCombination(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	ordering := ", in unspecified order"
	t.Run("yields independent snapshot copies for a known scope ID / resource ID / variant ID"+
		" combination "+ordering, func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			id1 := GenerateID(t)
			id2 := GenerateID(t)
			resourceID := GenerateID(t)

			refSnapshots := []*revision.ChronicleSnapshot{
				// snapshots that store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID) will
				// retrieve
				revision.NewChronicleSnapshot(id1, scopeID, resourceID, variantID, data),
				revision.NewChronicleSnapshot(id2, scopeID, resourceID, variantID, data),

				// snapshots that store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID) will
				// fail to retrieve
				// note: they are set later so their IDs sort last when sortSnapshots() is called
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,

				// snapshots that store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID) will
				// retrieve
				func() *revision.ChronicleSnapshot {
					result := GenerateChronicleSnapshotItem(t)
					result.ScopeID = scopeID
					result.ResourceID = resourceID
					result.VariantID = variantID
					return result
				}(),
			}
			refSnapshots[2] = GenerateChronicleSnapshotItem(t)
			refSnapshots[3] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.ScopeID = scopeID
				return result
			}()
			refSnapshots[4] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.ResourceID = resourceID
				return result
			}()
			refSnapshots[5] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.VariantID = variantID
				return result
			}()
			refSnapshots[6] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.ScopeID = scopeID
				result.ResourceID = resourceID
				return result
			}()
			refSnapshots[7] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.ScopeID = scopeID
				result.VariantID = variantID
				return result
			}()
			refSnapshots[8] = func() *revision.ChronicleSnapshot {
				result := GenerateChronicleSnapshotItem(t)
				result.ResourceID = resourceID
				result.VariantID = variantID
				return result
			}()
			for _, refSnapshot := range refSnapshots {
				AssertEqual(t, refSnapshot == nil, false).Critical() // make sure each nil slot was filled
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
			}

			addedSnapshots := make([]*revision.ChronicleSnapshot, len(refSnapshots))
			errAdds := make([]error, len(refSnapshots))
			for i := range refSnapshots {
				addedSnapshots[i] = refSnapshots[i].Clone()
				errAdds[i] = store.AddSnapshot(addedSnapshots[i])
			}
			defer CleanupStore(t, store, chronicleSnapshots2IDs(addedSnapshots))

			// store.GetByScopeAndResourceAndVariant() does not guarantee any particular ordering; so we sort
			// before comparison
			sortSnapshots := func(snapshots []*revision.ChronicleSnapshot) []*revision.ChronicleSnapshot {
				snapshots = slices.Clone(snapshots)
				slices.SortFunc(snapshots, func(a, b *revision.ChronicleSnapshot) int {
					return stdcmp.Compare(a.ID, b.ID)
				})
				return snapshots
			}

			refSorted := sortSnapshots(refSnapshots)
			addedSorted := sortSnapshots(addedSnapshots)
			for _, errAdd := range errAdds {
				AssertErrorIs(t, errAdd, nil).Critical()
			}

			for range ScenarioRepeatCount {
				gotSnapshots, errGet := store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID)
				gotSorted := sortSnapshots(gotSnapshots)

				AssertErrorIs(t, errGet, nil).Critical()
				AssertEqual(t, gotSorted, refSorted[:len(refSorted)-7], timeOpts...).Critical()
				AssertEqual(t, gotSorted, addedSorted[:len(addedSorted)-7], timeOpts...).Critical()

				for i, gotSnapshot := range gotSorted {
					refSnapshot, addedSnapshot := refSorted[i], addedSorted[i]

					assertAddedChronicleSnapshotPreserved(
						t, nil, nil, refSnapshot, addedSnapshot, gotSnapshot, timeOpts, preserveTimeLocation,
					)
					assertRetrievedChronicleSnapshotIndependent(t, addedSnapshot, gotSnapshot)

					// detect newly added fields so they can be accounted for in tests when necessary
					AssertFieldNamesEqual(t, gotSnapshot, []string{
						"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
					}).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctGetCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					snapshots, err := store.GetByScopeAndResourceAndVariant(
						snapshots[workerIdx].ScopeID,
						snapshots[workerIdx].ResourceID,
						snapshots[workerIdx].VariantID,
					)
					if err == nil && len(snapshots) == 1 {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctGetCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreMapsID(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for an ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			id1, id2 := fixture.id1, fixture.id2

			unknownID := GenerateID(t)

			testMapsUnknownID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsID(unknownID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()
				}
			}

			testMapsKnownID := func(store storage.ChronicleStore, id string) {
				for range ScenarioRepeatCount {
					got, err := store.MapsID(id)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testMapsUnknownID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testMapsUnknownID(store)
				testMapsKnownID(store, id1)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testMapsUnknownID(store)
				testMapsKnownID(store, id1)
				testMapsKnownID(store, id2)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctExistsCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					exists, err := store.MapsID(snapshots[workerIdx].ID)
					if err == nil && exists {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctExistsCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreMapsScopeID(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for a scope ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			unknownScopeID := GeneratePointerID(t)

			testMapsUnknownScopeID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()
				}
			}

			testMapsKnownScopeID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeID(scopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testMapsUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testMapsUnknownScopeID(store)
				testMapsKnownScopeID(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testMapsUnknownScopeID(store)
				testMapsKnownScopeID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctExistsCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					exists, err := store.MapsScopeID(snapshots[workerIdx].ScopeID)
					if err == nil && exists {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctExistsCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreMapsScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for a scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			resourceID := fixture.resourceID

			unknownScopeID := GeneratePointerID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GeneratePointerID(t)

			testMapsUnknownCombination := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					// 3 fields unknown

					got, err := store.MapsScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					// 2 fields unknown

					got, err = store.MapsScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, variantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					got, err = store.MapsScopeAndResourceAndVariant(
						unknownScopeID, resourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					got, err = store.MapsScopeAndResourceAndVariant(
						scopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					// 1 field unknown

					got, err = store.MapsScopeAndResourceAndVariant(unknownScopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					got, err = store.MapsScopeAndResourceAndVariant(scopeID, unknownResourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()

					got, err = store.MapsScopeAndResourceAndVariant(scopeID, resourceID, unknownVariantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()
				}
			}

			testMapsKnownCombination := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeAndResourceAndVariant(scopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testMapsUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testMapsUnknownCombination(store)
				testMapsKnownCombination(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testMapsUnknownCombination(store)
				testMapsKnownCombination(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctExistsCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					exists, err := store.MapsScopeAndResourceAndVariant(
						snapshots[workerIdx].ScopeID,
						snapshots[workerIdx].ResourceID,
						snapshots[workerIdx].VariantID,
					)
					if err == nil && exists {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctExistsCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, snapshot == snapshots[i], false).Critical()
				AssertEqual(t, snapshot, snapshots[i], timeOpts...).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreDeleteByID(t *testing.T, newStore func(t *testing.T) storage.ChronicleStore) {
	t.Run("deletes the snapshot corresponding to an ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			id1, id2 := fixture.id1, fixture.id2

			unknownID := GenerateID(t)

			testDeleteByUnknownID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.DeleteByID(unknownID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()
				}
			}

			testDeleteByKnownID := func(store storage.ChronicleStore, id string) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByID(id)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(1)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testDeleteByUnknownID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testDeleteByUnknownID(store)
				testDeleteByKnownID(store, id1)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testDeleteByUnknownID(store)
				testDeleteByKnownID(store, id1)
				testDeleteByKnownID(store, id2)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctDeleteCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					count, err := store.DeleteByID(snapshots[workerIdx].ID)
					if err == nil && count == 1 {
						atomic.AddInt64(&result, count)
					}
				})
				return result
			}()

			AssertEqual(t, correctDeleteCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				exists, err := store.MapsID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, exists, false).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreDeleteByScopeID(t *testing.T, newStore func(t *testing.T) storage.ChronicleStore) {
	t.Run("deletes the snapshots corresponding to a scope ID", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			unknownScopeID := GeneratePointerID(t)

			testDeleteByUnknownScopeID := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					got, err := store.DeleteByScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()
				}
			}

			testDeleteByKnownScopeID1 := func(store storage.ChronicleStore) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByScopeID(scopeID)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(1)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			testDeleteByKnownScopeID2 := func(store storage.ChronicleStore) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByScopeID(scopeID)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(2)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testDeleteByUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testDeleteByUnknownScopeID(store)
				testDeleteByKnownScopeID1(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testDeleteByUnknownScopeID(store)
				testDeleteByKnownScopeID2(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			matchCount := 0
			for _, snapshot := range snapshots {
				if ptr.EqualString(snapshot.ScopeID, scopeID) {
					matchCount++
				}
			}

			correctDeleteCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					count, err := store.DeleteByScopeID(snapshots[workerIdx].ScopeID)
					if err == nil && count == int64(matchCount) {
						atomic.AddInt64(&result, count)
					}
				})
				return result
			}()

			AssertEqual(t, correctDeleteCount, int64(matchCount)).Critical()
			for i := range WorkerCount {
				exists, err := store.MapsID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, exists, false).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func TestChronicleStoreDeleteByScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
) {
	t.Run("deletes the snapshots corresponding to a scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			fixture := setupChronicleStoreFixture(t, newStore, scopeID, variantID, data)

			resourceID := fixture.resourceID

			unknownScopeID := GeneratePointerID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GeneratePointerID(t)

			testDeleteByUnknownCombination := func(store storage.ChronicleStore) {
				for range ScenarioRepeatCount {
					// 3 fields unknown

					got, err := store.DeleteByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					// 2 fields unknown

					got, err = store.DeleteByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, variantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					got, err = store.DeleteByScopeAndResourceAndVariant(
						unknownScopeID, resourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					got, err = store.DeleteByScopeAndResourceAndVariant(
						scopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					// 1 field unknown

					got, err = store.DeleteByScopeAndResourceAndVariant(unknownScopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					got, err = store.DeleteByScopeAndResourceAndVariant(scopeID, unknownResourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()

					got, err = store.DeleteByScopeAndResourceAndVariant(scopeID, resourceID, unknownVariantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()
				}
			}

			testDeleteByKnownCombination1 := func(store storage.ChronicleStore) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByScopeAndResourceAndVariant(scopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(1)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			testDeleteByKnownCombination2 := func(store storage.ChronicleStore) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByScopeAndResourceAndVariant(scopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(2)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.ChronicleStore) {
				testDeleteByUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.ChronicleStore) {
				testDeleteByUnknownCombination(store)
				testDeleteByKnownCombination1(store)
			})

			fixture.withStoreN(func(store storage.ChronicleStore) {
				testDeleteByUnknownCombination(store)
				testDeleteByKnownCombination2(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(scopeID *string, variantID *string, data []byte) {
			store := newStore(t)

			snapshots := GenerateChronicleSnapshotsInStore(t, store, scopeID, variantID, data, WorkerCount)
			defer CleanupStore(t, store, chronicleSnapshots2IDs(snapshots))

			correctDeleteCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					count, err := store.DeleteByScopeAndResourceAndVariant(
						snapshots[workerIdx].ScopeID,
						snapshots[workerIdx].ResourceID,
						snapshots[workerIdx].VariantID,
					)
					if err == nil && count == 1 {
						atomic.AddInt64(&result, count)
					}
				})
				return result
			}()

			AssertEqual(t, correctDeleteCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				exists, err := store.MapsID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()
				AssertEqual(t, exists, false).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(GeneratePointerID(t), nil, nil)
		check(nil, GeneratePointerID(t), nil)
		check(nil, nil, GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), nil)
		check(GeneratePointerID(t), nil, GenerateBytes(t))
		check(nil, GeneratePointerID(t), GenerateBytes(t))
		check(GeneratePointerID(t), GeneratePointerID(t), GenerateBytes(t))
	})
}

func assertAddedChronicleSnapshotPreserved(
	t *testing.T,
	errAdd error,
	errGet error,
	refSnapshot *revision.ChronicleSnapshot,
	addedSnapshot *revision.ChronicleSnapshot,
	gotSnapshot *revision.ChronicleSnapshot,
	timeOpts []cmp.Option,
	preserveTimeLocation bool,
) {
	t.Helper()

	AssertErrorIs(t, errAdd, nil).Critical()

	AssertErrorIs(t, errGet, nil).Critical()
	AssertEqual(t, gotSnapshot, addedSnapshot, timeOpts...).Critical()
	AssertEqual(t, gotSnapshot, refSnapshot, timeOpts...).Critical()
	if preserveTimeLocation {
		AssertTimeLocationIs(t, gotSnapshot.CreatedAt, refSnapshot.CreatedAt.Location()).Critical()
	}
}

func assertRetrievedChronicleSnapshotIndependent(
	t *testing.T,
	addedSnapshot *revision.ChronicleSnapshot,
	gotSnapshot *revision.ChronicleSnapshot,
) {
	t.Helper()

	addedCopy := addedSnapshot.Clone()

	// mutate addedSnapshot

	time.Sleep(1 * time.Millisecond) // ensure addedSnapshot.CreatedAt has a new value
	addedSnapshot.ID = GenerateID(t)
	if addedSnapshot.ScopeID == nil {
		addedSnapshot.ScopeID = GeneratePointerID(t)
	} else {
		*addedSnapshot.ScopeID = GenerateID(t)
	}
	addedSnapshot.ResourceID = GenerateID(t)
	if addedSnapshot.VariantID == nil {
		addedSnapshot.VariantID = GeneratePointerID(t)
	} else {
		*addedSnapshot.VariantID = GenerateID(t)
	}
	if len(addedSnapshot.Data) == 0 { // empty or nil
		addedSnapshot.Data = GenerateBytes(t)
	} else {
		copy(addedSnapshot.Data, GenerateBytes(t))
	}
	addedSnapshot.CreatedAt = time.Now()

	// verify that changes to addedSnapshot are not reflected in gotSnapshot

	AssertNotEqual(t, addedSnapshot.ID, gotSnapshot.ID).Critical()
	AssertNotEqual(t, addedSnapshot.ScopeID, gotSnapshot.ScopeID).Critical()
	AssertNotEqual(t, addedSnapshot.ResourceID, gotSnapshot.ResourceID).Critical()
	AssertNotEqual(t, addedSnapshot.VariantID, gotSnapshot.VariantID).Critical()
	AssertNotEqual(t, addedSnapshot.Data, gotSnapshot.Data).Critical()
	AssertNotEqual(t, addedSnapshot.CreatedAt, gotSnapshot.CreatedAt).Critical()

	// restore addedSnapshot to its initial state

	*addedSnapshot = *addedCopy
}

func chronicleSnapshots2IDs(snapshots []*revision.ChronicleSnapshot) []string {
	var ids []string
	for i := range snapshots {
		if snapshots[i] != nil {
			ids = append(ids, snapshots[i].ID)
		}
	}
	return ids
}

type chronicleStoreFixtureData struct {
	id1        string
	id2        string
	resourceID string

	withStore0 func(fn func(store storage.ChronicleStore))
	withStore1 func(fn func(store storage.ChronicleStore))
	withStoreN func(fn func(store storage.ChronicleStore))
}

func setupChronicleStoreFixture(
	t *testing.T,
	newStore func(t *testing.T) storage.ChronicleStore,
	scopeID *string,
	variantID *string,
	data []byte,
) chronicleStoreFixtureData {
	id1 := GenerateID(t)
	id2 := GenerateID(t)
	resourceID := GenerateID(t)

	snapshot1 := revision.NewChronicleSnapshot(id1, scopeID, resourceID, variantID, data)
	snapshot2 := GenerateChronicleSnapshotItem(t)
	snapshot3 := revision.NewChronicleSnapshot(id2, scopeID, resourceID, variantID, data)

	return chronicleStoreFixtureData{
		id1:        id1,
		id2:        id2,
		resourceID: resourceID,

		// store0, store1 and storeN are created within their respective callbacks and cleaned up when the
		// callbacks return, so that the stores remain independent and do not affect one another. Indeed, as
		// implied by storage.ChronicleStore.CanShareState(), different store instances may share the same
		// underlying state when tested against a single storage backend (e.g. a database).

		withStore0: func(fn func(store storage.ChronicleStore)) {
			store0 := newStore(t) // empty store
			fn(store0)
		},

		withStore1: func(fn func(store storage.ChronicleStore)) {
			store1 := newStore(t) // store with one snapshot
			for i, snapshot := range []*revision.ChronicleSnapshot{snapshot1} {
				err := store1.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to store1: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, store1, []string{snapshot1.ID})
			fn(store1)
		},

		withStoreN: func(fn func(store storage.ChronicleStore)) {
			storeN := newStore(t) // store with several snapshots
			for i, snapshot := range []*revision.ChronicleSnapshot{snapshot1, snapshot2, snapshot3} {
				err := storeN.AddSnapshot(snapshot)
				if err != nil {
					t.Fatalf(
						"failed to add snapshot from index %d to storeN: %v",
						i,
						failures.AsErrorWithSemantics(err),
					)
				}
			}
			defer CleanupStore(t, storeN, []string{snapshot1.ID, snapshot2.ID, snapshot3.ID})
			fn(storeN)
		},
	}
}
