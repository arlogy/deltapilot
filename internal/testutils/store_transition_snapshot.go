package testutils

import (
	"bytes"
	stdcmp "cmp"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Notes on this file's test flow.
// - Some assertions establish prerequisites for other assertions.
// - Some flows are repeated ScenarioRepeatCount times for consistency.
// - We therefore use Critical() to prevent cascading errors when a prerequisite fails.

func GenerateTransitionSnapshotItem(t *testing.T) *revision.TransitionSnapshot {
	t.Helper()

	id := GenerateID(t)
	scopeID := GenerateID(t)
	resourceID := GenerateID(t)
	variantID := GenerateID(t)
	baselineData := GenerateBytes(t)
	targetData := GenerateBytes(t)

	// return a snapshot with all nullable fields intentionally set to non-null values
	return revision.NewTransitionSnapshot(id, scopeID, resourceID, variantID, baselineData, targetData)
}

func GenerateTransitionSnapshotSlice(
	t *testing.T,
	scopeID string,
	baselineData []byte,
	targetData []byte,
	includeNil bool,
	count int,
) []*revision.TransitionSnapshot {
	t.Helper()

	newSnapshot1 := func(t *testing.T) *revision.TransitionSnapshot {
		id := GenerateID(t)
		resourceID := GenerateID(t)
		variantID := GenerateID(t)
		return revision.NewTransitionSnapshot(id, scopeID, resourceID, variantID, baselineData, targetData)
	}

	newSnapshot2 := func(t *testing.T) *revision.TransitionSnapshot {
		return GenerateTransitionSnapshotItem(t)
	}

	return GenerateSnapshots(t, newSnapshot1, newSnapshot2, includeNil, count)
}

func GenerateTransitionSnapshotsInStore(
	t *testing.T,
	store storage.TransitionStore,
	scopeID string,
	baselineData []byte,
	targetData []byte,
	count int,
) []*revision.TransitionSnapshot {
	t.Helper()

	snapshots := GenerateTransitionSnapshotSlice(t, scopeID, baselineData, targetData, false, count)

	AddSnapshotsToStore(t, store, snapshots)

	return snapshots
}

func TestTransitionStoreCanShareState(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	stateSharable bool,
) {
	t.Run("returns "+fmt.Sprintf("%t", stateSharable), func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			testStateSharing := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got := store.CanShareState()
					AssertEqual(t, got, stateSharable).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testStateSharing(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testStateSharing(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testStateSharing(store)
			})

			func() {
				newSnapshot := func(t *testing.T) (string, *revision.TransitionSnapshot) {
					id := GenerateID(t)
					scopeID := GenerateID(t)
					resourceID := GenerateID(t)
					variantID := GenerateID(t)
					return id, revision.NewTransitionSnapshot(
						id, scopeID, resourceID, variantID, baselineData, targetData,
					)
				}

				CheckStoreStateSharing(t, newStore, newSnapshot, stateSharable)
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreAddSnapshot(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	duplicateIDErrors func(snapshot *revision.TransitionSnapshot) (error, error),
	duplicateScopeAndResourceAndVariantErrors func(snapshot *revision.TransitionSnapshot) (error, error),
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects a nil snapshot", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			testAddNilSnapshot := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					err := store.AddSnapshot(nil)
					AssertErrorIs(t, err, storage.ErrSnapshotRequired).Critical()
					AssertErrorMessage(t, err, storage.WrapSnapshotRequired().Error()).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testAddNilSnapshot(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testAddNilSnapshot(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testAddNilSnapshot(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("rejects a snapshot with a duplicate ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			defer CleanupStore(t, store, []string{id})

			snapshot := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)
			err := store.AddSnapshot(snapshot)
			AssertErrorIs(t, err, nil).Critical()

			func() {
				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateIDErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()

			func() {
				snapshot := GenerateTransitionSnapshotItem(t)
				snapshot.ID = id

				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateIDErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("rejects a snapshot with a duplicate scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			defer CleanupStore(t, store, []string{id})

			snapshot := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)
			err := store.AddSnapshot(snapshot)
			AssertErrorIs(t, err, nil).Critical()

			func() {
				snapshot := revision.NewTransitionSnapshot(
					GenerateID(t), scopeID, resourceID, variantID, baselineData, targetData,
				)
				defer CleanupStore(t, store, []string{snapshot.ID})

				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateScopeAndResourceAndVariantErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()

			func() {
				snapshot := GenerateTransitionSnapshotItem(t)
				snapshot.ScopeID = scopeID
				snapshot.ResourceID = resourceID
				snapshot.VariantID = variantID
				defer CleanupStore(t, store, []string{snapshot.ID})

				for range ScenarioRepeatCount {
					err := store.AddSnapshot(snapshot)
					wrappedErr, returnedErr := duplicateScopeAndResourceAndVariantErrors(snapshot)
					CheckStoreOutputError(t, err, wrappedErr, returnedErr)
				}
			}()
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("stores an independent copy of the snapshot when accepted", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			for range ScenarioRepeatCount {
				id := GenerateID(t)
				scopeID := GenerateID(t)
				resourceID := GenerateID(t)
				variantID := GenerateID(t)

				refSnapshot := revision.NewTransitionSnapshot(
					id, scopeID, resourceID, variantID, baselineData, targetData,
				)
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
				refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())

				addedSnapshot := refSnapshot.Clone()
				errAdd := store.AddSnapshot(addedSnapshot)
				defer CleanupStore(t, store, []string{id})

				gotSnapshot, errGet := store.GetByID(id)

				assertAddedTransitionSnapshotPreserved(
					t, errAdd, errGet, refSnapshot, addedSnapshot, gotSnapshot, timeOpts,
					preserveTimeLocation,
				)
				assertRetrievedTransitionSnapshotIndependent(t, addedSnapshot, gotSnapshot, nil)

				assertRetrievedTransitionSnapshotIndependent(
					t, addedSnapshot, nil, func(t *testing.T) *revision.TransitionSnapshot {
						snapshot, err := store.GetByID(id)
						AssertErrorIs(t, err, nil).Critical()
						return snapshot
					},
				)

				// detect newly added fields so they can be accounted for in tests when necessary
				AssertFieldNamesEqual(t, gotSnapshot, []string{
					"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
					"UpdatedAt",
				}).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotSlice(
				t, scopeID, baselineData, targetData, true, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreGetByID(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects an unknown snapshot ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownID := GenerateID(t)

			testGetByUnknownID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.GetByID(unknownID)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(t, err, storage.WrapSnapshotNotFoundByID(unknownID).Error()).Critical()
					AssertEqual(t, got == nil, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testGetByUnknownID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testGetByUnknownID(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testGetByUnknownID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("yields an independent snapshot copy for a known ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)

			refSnapshot := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)
			refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
			refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())

			addedSnapshot := refSnapshot.Clone()
			errAdd := store.AddSnapshot(addedSnapshot)
			defer CleanupStore(t, store, []string{id})

			for range ScenarioRepeatCount {
				gotSnapshot, errGet := store.GetByID(id)

				assertAddedTransitionSnapshotPreserved(
					t, errAdd, errGet, refSnapshot, addedSnapshot, gotSnapshot, timeOpts,
					preserveTimeLocation,
				)
				assertRetrievedTransitionSnapshotIndependent(t, addedSnapshot, gotSnapshot, nil)

				gotSnapshot2, errGet2 := store.GetByID(id)
				AssertErrorIs(t, errGet2, nil).Critical()
				AssertEqual(t, gotSnapshot2, gotSnapshot).Critical()
				assertRetrievedTransitionSnapshotIndependent(t, gotSnapshot, gotSnapshot2, nil)

				// detect newly added fields so they can be accounted for in tests when necessary
				AssertFieldNamesEqual(t, gotSnapshot, []string{
					"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
					"UpdatedAt",
				}).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreGetByScopeID(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("yields an empty snapshot slice for an unknown scope ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)

			testGetByUnknownScopeID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.GetByScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, len(got), 0).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testGetByUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testGetByUnknownScopeID(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testGetByUnknownScopeID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	ordering := ", in unspecified order"
	t.Run("yields independent snapshot copies for a known scope ID"+ordering, func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id1 := GenerateID(t)
			id2 := GenerateID(t)
			id3 := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)

			refSnapshots := []*revision.TransitionSnapshot{
				// snapshots that store.GetByScopeID(scopeID) will retrieve
				revision.NewTransitionSnapshot(id1, scopeID, resourceID, variantID, baselineData, targetData),
				revision.NewTransitionSnapshot(
					id2, scopeID, GenerateID(t), variantID, baselineData, targetData,
				),
				revision.NewTransitionSnapshot(
					id3, scopeID, resourceID, GenerateID(t), baselineData, targetData,
				),

				// snapshots that store.GetByScopeID(scopeID) will fail to retrieve
				// note: they are set later so their IDs sort last when sortSnapshots() is called
				nil,

				// snapshots that store.GetByScopeID(scopeID) will retrieve
				func() *revision.TransitionSnapshot {
					result := GenerateTransitionSnapshotItem(t)
					result.ScopeID = scopeID
					return result
				}(),
			}
			refSnapshots[3] = GenerateTransitionSnapshotItem(t)
			for _, refSnapshot := range refSnapshots {
				AssertEqual(t, refSnapshot == nil, false).Critical() // make sure each nil slot was filled
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
				refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())
			}

			addedSnapshots := make([]*revision.TransitionSnapshot, len(refSnapshots))
			errAdds := make([]error, len(refSnapshots))
			for i := range refSnapshots {
				addedSnapshots[i] = refSnapshots[i].Clone()
				errAdds[i] = store.AddSnapshot(addedSnapshots[i])
			}
			defer CleanupStore(t, store, transitionSnapshots2IDs(addedSnapshots))

			// store.GetByScopeID() does not guarantee any particular ordering; so we sort before comparison
			sortSnapshots := func(snapshots []*revision.TransitionSnapshot) []*revision.TransitionSnapshot {
				snapshots = slices.Clone(snapshots)
				slices.SortFunc(snapshots, func(a, b *revision.TransitionSnapshot) int {
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

					assertAddedTransitionSnapshotPreserved(
						t, nil, nil, refSnapshot, addedSnapshot, gotSnapshot, timeOpts, preserveTimeLocation,
					)
					assertRetrievedTransitionSnapshotIndependent(t, addedSnapshot, gotSnapshot, nil)

					// detect newly added fields so they can be accounted for in tests when necessary
					AssertFieldNamesEqual(t, gotSnapshot, []string{
						"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
						"UpdatedAt",
					}).Critical()
				}

				gotSnapshots2, errGet2 := store.GetByScopeID(scopeID)
				gotSorted2 := sortSnapshots(gotSnapshots2)

				AssertErrorIs(t, errGet2, nil).Critical()
				AssertEqual(t, gotSorted2, gotSorted).Critical()
				for i, gotSnapshot := range gotSorted {
					assertRetrievedTransitionSnapshotIndependent(t, gotSnapshot, gotSorted2[i], nil)
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

			matchCount := 0
			for _, snapshot := range snapshots {
				if snapshot.ScopeID == scopeID {
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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreGetByScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects an unknown scope ID / resource ID / variant ID combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GenerateID(t)

			testGetByUnknownCombination := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					// 3 fields unknown

					got, err := store.GetByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&unknownScopeID, unknownResourceID, &unknownVariantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					// 2 fields unknown

					got, err = store.GetByScopeAndResourceAndVariant(
						unknownScopeID, unknownResourceID, variantID,
					)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&unknownScopeID, unknownResourceID, &variantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(
						unknownScopeID, resourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&unknownScopeID, resourceID, &unknownVariantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(
						scopeID, unknownResourceID, unknownVariantID,
					)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&scopeID, unknownResourceID, &unknownVariantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					// 1 field unknown

					got, err = store.GetByScopeAndResourceAndVariant(unknownScopeID, resourceID, variantID)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&unknownScopeID, resourceID, &variantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(scopeID, unknownResourceID, variantID)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&scopeID, unknownResourceID, &variantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()

					got, err = store.GetByScopeAndResourceAndVariant(scopeID, resourceID, unknownVariantID)
					AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
					AssertErrorMessage(
						t, err,
						storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
							&scopeID, resourceID, &unknownVariantID,
						).Error(),
					).Critical()
					AssertEqual(t, got == nil, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testGetByUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testGetByUnknownCombination(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testGetByUnknownCombination(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("yields an independent snapshot copy for a known scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id1 := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)

			refSnapshots := []*revision.TransitionSnapshot{
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
				revision.NewTransitionSnapshot(id1, scopeID, resourceID, variantID, baselineData, targetData),
			}
			refSnapshots[0] = GenerateTransitionSnapshotItem(t)
			refSnapshots[1] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.ScopeID = scopeID
				return result
			}()
			refSnapshots[2] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.ResourceID = resourceID
				return result
			}()
			refSnapshots[3] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.VariantID = variantID
				return result
			}()
			refSnapshots[4] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.ScopeID = scopeID
				result.ResourceID = resourceID
				return result
			}()
			refSnapshots[5] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.ScopeID = scopeID
				result.VariantID = variantID
				return result
			}()
			refSnapshots[6] = func() *revision.TransitionSnapshot {
				result := GenerateTransitionSnapshotItem(t)
				result.ResourceID = resourceID
				result.VariantID = variantID
				return result
			}()
			for _, refSnapshot := range refSnapshots {
				AssertEqual(t, refSnapshot == nil, false).Critical() // make sure each nil slot was filled
				refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
				refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())
			}

			addedSnapshots := make([]*revision.TransitionSnapshot, len(refSnapshots))
			errAdds := make([]error, len(refSnapshots))
			for i := range refSnapshots {
				addedSnapshots[i] = refSnapshots[i].Clone()
				errAdds[i] = store.AddSnapshot(addedSnapshots[i])
			}
			defer CleanupStore(t, store, transitionSnapshots2IDs(addedSnapshots))

			for _, errAdd := range errAdds {
				AssertErrorIs(t, errAdd, nil).Critical()
			}

			for range ScenarioRepeatCount {
				gotSnapshot, errGet := store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID)

				refSnapshot, addedSnapshot := refSnapshots[7], addedSnapshots[7]

				AssertErrorIs(t, errGet, nil).Critical()
				AssertEqual(t, gotSnapshot, refSnapshot, timeOpts...).Critical()
				AssertEqual(t, gotSnapshot, addedSnapshot, timeOpts...).Critical()

				assertAddedTransitionSnapshotPreserved(
					t, nil, nil, refSnapshot, addedSnapshot, gotSnapshot, timeOpts, preserveTimeLocation,
				)
				assertRetrievedTransitionSnapshotIndependent(t, addedSnapshot, gotSnapshot, nil)

				gotSnapshot2, errGet2 := store.GetByScopeAndResourceAndVariant(scopeID, resourceID, variantID)
				AssertErrorIs(t, errGet2, nil).Critical()
				AssertEqual(t, gotSnapshot2, gotSnapshot).Critical()
				assertRetrievedTransitionSnapshotIndependent(t, gotSnapshot, gotSnapshot2, nil)

				// detect newly added fields so they can be accounted for in tests when necessary
				AssertFieldNamesEqual(t, gotSnapshot, []string{
					"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
					"UpdatedAt",
				}).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

			correctGetCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					snapshot, err := store.GetByScopeAndResourceAndVariant(
						snapshots[workerIdx].ScopeID,
						snapshots[workerIdx].ResourceID,
						snapshots[workerIdx].VariantID,
					)
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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreMapsID(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for an ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			recordedIDs := fixture.recordedIDs

			unknownID := GenerateID(t)

			testMapsUnknownID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsID(unknownID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()
				}
			}

			testMapsKnownID := func(store storage.TransitionStore, id string) {
				for range ScenarioRepeatCount {
					got, err := store.MapsID(id)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testMapsUnknownID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testMapsUnknownID(store)
				testMapsKnownID(store, recordedIDs[0])
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testMapsUnknownID(store)
				for _, id := range recordedIDs {
					testMapsKnownID(store, id)
				}
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreMapsScopeID(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for a scope ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)

			testMapsUnknownScopeID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, false).Critical()
				}
			}

			testMapsKnownScopeID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeID(scopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testMapsUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testMapsUnknownScopeID(store)
				testMapsKnownScopeID(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testMapsUnknownScopeID(store)
				testMapsKnownScopeID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreMapsScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("indicates whether a snapshot exists for a scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GenerateID(t)

			testMapsUnknownCombination := func(store storage.TransitionStore) {
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

			testMapsKnownCombination := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.MapsScopeAndResourceAndVariant(scopeID, resourceID, variantID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, true).Critical()
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testMapsUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testMapsUnknownCombination(store)
				testMapsKnownCombination(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testMapsUnknownCombination(store)
				testMapsKnownCombination(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionApplyTransitionForID(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects an unknown snapshot ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownID := GenerateID(t)

			testApplyTransitionForUnknownID := func(store storage.TransitionStore) {
				for _, newTargetData := range [][]byte{targetData, nil, GenerateBytes(t)} {
					for range ScenarioRepeatCount {
						err := store.ApplyTransitionForID(unknownID, newTargetData)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(t, err, storage.WrapSnapshotNotFoundByID(unknownID).Error()).
							Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownID(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownID(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("applies snapshot transition for a known ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			defer CleanupStore(t, store, []string{id})

			refSnapshot := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)
			refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
			refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())

			addedSnapshot := refSnapshot.Clone()
			err := store.AddSnapshot(addedSnapshot)
			AssertErrorIs(t, err, nil).Critical()

			for _, newTargetData := range [][]byte{targetData, nil, GenerateBytes(t)} {
				for range ScenarioRepeatCount {
					newTargetSource := bytes.Clone(newTargetData)

					timeBeforeTransition := time.Now()
					time.Sleep(1 * time.Millisecond) // ensure gotSnapshot.UpdatedAt has a new value
					err1 := store.ApplyTransitionForID(id, newTargetData)
					AssertErrorIs(t, err1, nil).Critical()

					gotSnapshot, err2 := store.GetByID(id)
					AssertErrorIs(t, err2, nil).Critical()

					timeOpts2 := append(slices.Clone(timeOpts), cmpopts.IgnoreFields(
						revision.TransitionSnapshot{}, "TargetData", "UpdatedAt",
					))
					AssertEqual(t, gotSnapshot, refSnapshot, timeOpts2...).Critical()
					AssertEqual(t, gotSnapshot.TargetData, newTargetSource).Critical()
					AssertSlicesIndependent(t, gotSnapshot.TargetData, newTargetData).Critical()
					AssertTimeAfter(t, gotSnapshot.UpdatedAt, timeBeforeTransition).Critical()
					if preserveTimeLocation {
						AssertTimeLocationIs(t, gotSnapshot.CreatedAt, refSnapshot.CreatedAt.Location()).
							Critical()
						AssertTimeLocationIs(t, gotSnapshot.UpdatedAt, time.UTC).Critical()
					}

					// detect newly added fields so they can be accounted for in tests when necessary
					AssertFieldNamesEqual(t, gotSnapshot, []string{
						"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
						"UpdatedAt",
					}).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

			correctApplyCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					time.Sleep(1 * time.Millisecond) // ensure the snapshot's UpdatedAt has a new value
					err := store.ApplyTransitionForID(snapshots[workerIdx].ID, GenerateBytes(t))
					if err == nil {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctApplyCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()

				timeOpts2 := append(slices.Clone(timeOpts), cmpopts.IgnoreFields(
					revision.TransitionSnapshot{}, "TargetData", "UpdatedAt",
				))
				AssertEqual(t, snapshot, snapshots[i], timeOpts2...).Critical()
				AssertNotEqual(t, snapshot.TargetData, snapshots[i].TargetData).Critical()
				AssertTimeAfter(t, snapshot.UpdatedAt, snapshots[i].UpdatedAt).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionApplyTransitionForScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	timeTolerance time.Duration,
	preserveTimeLocation bool,
) {
	timeOpts := []cmp.Option{}
	if timeTolerance > 0 {
		timeOpts = append(timeOpts, cmpopts.EquateApproxTime(timeTolerance))
	}

	t.Run("rejects an unknown scope ID / resource ID / variant ID combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GenerateID(t)

			testApplyTransitionForUnknownCombination := func(store storage.TransitionStore) {
				for _, newTargetData := range [][]byte{targetData, nil, GenerateBytes(t)} {
					for range ScenarioRepeatCount {
						// 3 fields unknown

						err := store.ApplyTransitionForScopeAndResourceAndVariant(
							unknownScopeID, unknownResourceID, unknownVariantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&unknownScopeID, unknownResourceID, &unknownVariantID,
							).Error(),
						).Critical()

						// 2 fields unknown

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							unknownScopeID, unknownResourceID, variantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&unknownScopeID, unknownResourceID, &variantID,
							).Error(),
						).Critical()

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							unknownScopeID, resourceID, unknownVariantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&unknownScopeID, resourceID, &unknownVariantID,
							).Error(),
						).Critical()

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							scopeID, unknownResourceID, unknownVariantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&scopeID, unknownResourceID, &unknownVariantID,
							).Error(),
						).Critical()

						// 1 field unknown

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							unknownScopeID, resourceID, variantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&unknownScopeID, resourceID, &variantID,
							).Error(),
						).Critical()

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							scopeID, unknownResourceID, variantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&scopeID, unknownResourceID, &variantID,
							).Error(),
						).Critical()

						err = store.ApplyTransitionForScopeAndResourceAndVariant(
							scopeID, resourceID, unknownVariantID, newTargetData,
						)
						AssertErrorIs(t, err, storage.ErrSnapshotRetrieval).Critical()
						AssertErrorMessage(
							t, err,
							storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(
								&scopeID, resourceID, &unknownVariantID,
							).Error(),
						).Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownCombination(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testApplyTransitionForUnknownCombination(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("applies snapshot transition for a known scope ID / resource ID / variant ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			id := GenerateID(t)
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			defer CleanupStore(t, store, []string{id})

			refSnapshot := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)
			refSnapshot.CreatedAt = refSnapshot.CreatedAt.In(GenerateTimeZone())
			refSnapshot.UpdatedAt = refSnapshot.UpdatedAt.In(GenerateTimeZone())

			addedSnapshot := refSnapshot.Clone()
			err := store.AddSnapshot(addedSnapshot)
			AssertErrorIs(t, err, nil).Critical()

			for _, newTargetData := range [][]byte{targetData, nil, GenerateBytes(t)} {
				for range ScenarioRepeatCount {
					newTargetSource := bytes.Clone(newTargetData)

					timeBeforeTransition := time.Now()
					time.Sleep(1 * time.Millisecond) // ensure gotSnapshot.UpdatedAt has a new value
					err1 := store.ApplyTransitionForScopeAndResourceAndVariant(
						scopeID, resourceID, variantID, newTargetData,
					)
					AssertErrorIs(t, err1, nil).Critical()

					gotSnapshot, err2 := store.GetByID(id)
					AssertErrorIs(t, err2, nil).Critical()

					timeOpts2 := append(slices.Clone(timeOpts), cmpopts.IgnoreFields(
						revision.TransitionSnapshot{}, "TargetData", "UpdatedAt",
					))
					AssertEqual(t, gotSnapshot, refSnapshot, timeOpts2...).Critical()
					AssertEqual(t, gotSnapshot.TargetData, newTargetSource).Critical()
					AssertSlicesIndependent(t, gotSnapshot.TargetData, newTargetData).Critical()
					AssertTimeAfter(t, gotSnapshot.UpdatedAt, timeBeforeTransition).Critical()
					if preserveTimeLocation {
						AssertTimeLocationIs(t, gotSnapshot.CreatedAt, refSnapshot.CreatedAt.Location()).
							Critical()
						AssertTimeLocationIs(t, gotSnapshot.UpdatedAt, time.UTC).Critical()
					}

					// detect newly added fields so they can be accounted for in tests when necessary
					AssertFieldNamesEqual(t, gotSnapshot, []string{
						"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
						"UpdatedAt",
					}).Critical()
				}
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

			correctApplyCount := func() int64 {
				var result int64
				RunConcurrently(t, func(workerIdx int) {
					time.Sleep(1 * time.Millisecond) // ensure the snapshot's UpdatedAt has a new value
					err := store.ApplyTransitionForScopeAndResourceAndVariant(
						snapshots[workerIdx].ScopeID,
						snapshots[workerIdx].ResourceID,
						snapshots[workerIdx].VariantID,
						GenerateBytes(t),
					)
					if err == nil {
						atomic.AddInt64(&result, 1)
					}
				})
				return result
			}()

			AssertEqual(t, correctApplyCount, int64(WorkerCount)).Critical()
			for i := range WorkerCount {
				snapshot, err := store.GetByID(snapshots[i].ID)
				AssertErrorIs(t, err, nil).Critical()

				timeOpts2 := append(slices.Clone(timeOpts), cmpopts.IgnoreFields(
					revision.TransitionSnapshot{}, "TargetData", "UpdatedAt",
				))
				AssertEqual(t, snapshot, snapshots[i], timeOpts2...).Critical()
				AssertNotEqual(t, snapshot.TargetData, snapshots[i].TargetData).Critical()
				AssertTimeAfter(t, snapshot.UpdatedAt, snapshots[i].UpdatedAt).Critical()
			}
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreDeleteByID(t *testing.T, newStore func(t *testing.T) storage.TransitionStore) {
	t.Run("deletes the snapshot corresponding to an ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			recordedIDs := fixture.recordedIDs

			unknownID := GenerateID(t)

			testDeleteByUnknownID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.DeleteByID(unknownID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()
				}
			}

			testDeleteByKnownID := func(store storage.TransitionStore, id string) {
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

			fixture.withStore0(func(store storage.TransitionStore) {
				testDeleteByUnknownID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testDeleteByUnknownID(store)
				testDeleteByKnownID(store, recordedIDs[0])
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testDeleteByUnknownID(store)
				for _, id := range recordedIDs {
					testDeleteByKnownID(store, id)
				}
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreDeleteByScopeID(t *testing.T, newStore func(t *testing.T) storage.TransitionStore) {
	t.Run("deletes the snapshots corresponding to a scope ID", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)

			testDeleteByUnknownScopeID := func(store storage.TransitionStore) {
				for range ScenarioRepeatCount {
					got, err := store.DeleteByScopeID(unknownScopeID)
					AssertErrorIs(t, err, nil).Critical()
					AssertEqual(t, got, int64(0)).Critical()
				}
			}

			testDeleteByKnownScopeID1 := func(store storage.TransitionStore) {
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

			testDeleteByKnownScopeID2 := func(store storage.TransitionStore) {
				for i := range ScenarioRepeatCount {
					got, err := store.DeleteByScopeID(scopeID)
					AssertErrorIs(t, err, nil).Critical()
					if i == 0 {
						AssertEqual(t, got, int64(3)).Critical()
					} else {
						AssertEqual(t, got, int64(0)).Critical()
					}
				}
			}

			fixture.withStore0(func(store storage.TransitionStore) {
				testDeleteByUnknownScopeID(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testDeleteByUnknownScopeID(store)
				testDeleteByKnownScopeID1(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testDeleteByUnknownScopeID(store)
				testDeleteByKnownScopeID2(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

			matchCount := 0
			for _, snapshot := range snapshots {
				if snapshot.ScopeID == scopeID {
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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func TestTransitionStoreDeleteByScopeAndResourceAndVariant(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
) {
	t.Run("deletes the snapshots corresponding to a scope ID / resource ID / variant ID"+
		" combination", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			scopeID := GenerateID(t)
			resourceID := GenerateID(t)
			variantID := GenerateID(t)
			fixture := transitionStoreFixture(
				t, newStore, scopeID, resourceID, variantID, baselineData, targetData,
			)

			unknownScopeID := GenerateID(t)
			unknownResourceID := GenerateID(t)
			unknownVariantID := GenerateID(t)

			testDeleteByUnknownCombination := func(store storage.TransitionStore) {
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

			testDeleteByKnownCombination1 := func(store storage.TransitionStore) {
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

			testDeleteByKnownCombination2 := func(store storage.TransitionStore) {
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

			fixture.withStore0(func(store storage.TransitionStore) {
				testDeleteByUnknownCombination(store)
			})

			fixture.withStore1(func(store storage.TransitionStore) {
				testDeleteByUnknownCombination(store)
				testDeleteByKnownCombination1(store)
			})

			fixture.withStoreN(func(store storage.TransitionStore) {
				testDeleteByUnknownCombination(store)
				testDeleteByKnownCombination2(store)
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})

	t.Run("handles concurrent calls safely", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping concurrency test")
		}

		check := func(baselineData []byte, targetData []byte) {
			store := newStore(t)

			scopeID := GenerateID(t)
			snapshots := GenerateTransitionSnapshotsInStore(
				t, store, scopeID, baselineData, targetData, WorkerCount,
			)
			defer CleanupStore(t, store, transitionSnapshots2IDs(snapshots))

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
		check(nil, nil)
		check(nil, GenerateBytes(t))
		check(GenerateBytes(t), nil)
		check(GenerateBytes(t), GenerateBytes(t))
	})
}

func assertAddedTransitionSnapshotPreserved(
	t *testing.T,
	errAdd error,
	errGet error,
	refSnapshot *revision.TransitionSnapshot,
	addedSnapshot *revision.TransitionSnapshot,
	gotSnapshot *revision.TransitionSnapshot,
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
		AssertTimeLocationIs(t, gotSnapshot.UpdatedAt, refSnapshot.UpdatedAt.Location()).Critical()
	}
}

func assertRetrievedTransitionSnapshotIndependent(
	t *testing.T,
	addedSnapshot *revision.TransitionSnapshot,
	gotSnapshot *revision.TransitionSnapshot,
	fetchSnapshot func(t *testing.T) *revision.TransitionSnapshot,
) {
	t.Helper()

	addedCopy := addedSnapshot.Clone()

	// mutate addedSnapshot

	addedSnapshot.ID = GenerateID(t)
	addedSnapshot.ScopeID = GenerateID(t)
	addedSnapshot.ResourceID = GenerateID(t)
	addedSnapshot.VariantID = GenerateID(t)
	if len(addedSnapshot.BaselineData) == 0 { // empty or nil
		addedSnapshot.BaselineData = GenerateBytes(t)
	} else {
		copy(addedSnapshot.BaselineData, GenerateBytes(t))
	}
	if len(addedSnapshot.TargetData) == 0 { // empty or nil
		addedSnapshot.TargetData = GenerateBytes(t)
	} else {
		copy(addedSnapshot.TargetData, GenerateBytes(t))
	}
	time.Sleep(1 * time.Millisecond) // ensure addedSnapshot.CreatedAt has a new value
	addedSnapshot.CreatedAt = time.Now()
	time.Sleep(1 * time.Millisecond) // ensure addedSnapshot.UpdatedAt has a new value
	addedSnapshot.UpdatedAt = time.Now()

	// verify that changes to addedSnapshot are not reflected in gotSnapshot

	if fetchSnapshot != nil { // intended for setting gotSnapshot after mutating addedSnapshot
		gotSnapshot = fetchSnapshot(t)
	}

	AssertNotEqual(t, addedSnapshot.ID, gotSnapshot.ID).Critical()
	AssertNotEqual(t, addedSnapshot.ScopeID, gotSnapshot.ScopeID).Critical()
	AssertNotEqual(t, addedSnapshot.ResourceID, gotSnapshot.ResourceID).Critical()
	AssertNotEqual(t, addedSnapshot.VariantID, gotSnapshot.VariantID).Critical()
	AssertNotEqual(t, addedSnapshot.BaselineData, gotSnapshot.BaselineData).Critical()
	AssertNotEqual(t, addedSnapshot.TargetData, gotSnapshot.TargetData).Critical()
	AssertNotEqual(t, addedSnapshot.CreatedAt, gotSnapshot.CreatedAt).Critical()
	AssertNotEqual(t, addedSnapshot.UpdatedAt, gotSnapshot.UpdatedAt).Critical()

	// restore addedSnapshot to its initial state

	*addedSnapshot = *addedCopy
}

func transitionSnapshots2IDs(snapshots []*revision.TransitionSnapshot) []string {
	var ids []string
	for i := range snapshots {
		if snapshots[i] != nil {
			ids = append(ids, snapshots[i].ID)
		}
	}
	return ids
}

func transitionStoreFixture(
	t *testing.T,
	newStore func(t *testing.T) storage.TransitionStore,
	scopeID string,
	resourceID string,
	variantID string,
	baselineData []byte,
	targetData []byte,
) storeFixtureData[storage.TransitionStore] {
	newSnapshots := func(t *testing.T) ([]string, []*revision.TransitionSnapshot) {
		id1 := GenerateID(t)
		id2 := GenerateID(t)
		id3 := GenerateID(t)
		id4 := GenerateID(t)
		snapshots := []*revision.TransitionSnapshot{
			revision.NewTransitionSnapshot(id1, scopeID, resourceID, variantID, baselineData, targetData),
			revision.NewTransitionSnapshot(
				id2, GenerateID(t), resourceID, variantID, baselineData, targetData,
			),
			revision.NewTransitionSnapshot(id3, scopeID, GenerateID(t), variantID, baselineData, targetData),
			revision.NewTransitionSnapshot(id4, scopeID, resourceID, GenerateID(t), baselineData, targetData),
			GenerateTransitionSnapshotItem(t),
		}
		return transitionSnapshots2IDs(snapshots), snapshots
	}

	return NewStoreFixture(t, newStore, newSnapshots)
}
