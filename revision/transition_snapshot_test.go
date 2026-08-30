package revision_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/revision"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewTransitionSnapshot(t *testing.T) {
	t.Run("creates a new snapshot", func(t *testing.T) {
		check := func(baselineData []byte, targetData []byte) {
			id := testutils.GenerateID(t)
			scopeID := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)
			variantID := testutils.GenerateID(t)

			baselineSource := bytes.Clone(baselineData)
			targetSource := bytes.Clone(targetData)

			timeBeforeSnapshotCreation := time.Now()
			time.Sleep(1 * time.Millisecond)
			got := revision.NewTransitionSnapshot(
				id, scopeID, resourceID, variantID, baselineData, targetData,
			)

			want := &revision.TransitionSnapshot{
				ID:           id,
				ScopeID:      scopeID,
				ResourceID:   resourceID,
				VariantID:    variantID,
				BaselineData: baselineSource,
				TargetData:   targetSource,
			}

			testutils.AssertEqual(t, got, want, cmpopts.IgnoreFields(
				revision.TransitionSnapshot{}, "CreatedAt", "UpdatedAt",
			))

			testutils.AssertTimeAfter(t, got.CreatedAt, timeBeforeSnapshotCreation)
			testutils.AssertTimeLocationIs(t, got.CreatedAt, time.UTC)

			testutils.AssertEqual(t, got.UpdatedAt, got.CreatedAt)
			testutils.AssertTimeLocationIs(t, got.UpdatedAt, time.UTC)

			testutils.AssertSlicesIndependent(t, got.BaselineData, baselineData)
			testutils.AssertSlicesIndependent(t, got.TargetData, targetData)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, got, []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
				"UpdatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil)
		check(nil, testutils.GenerateBytes(t))
		check(testutils.GenerateBytes(t), nil)
		check(testutils.GenerateBytes(t), testutils.GenerateBytes(t))
	})
}

func TestPlanTransitionTo(t *testing.T) {
	t.Run("builds snapshot transition plan", func(t *testing.T) {
		check := func(targetData []byte) {
			targetSource := bytes.Clone(targetData)

			timeBeforeTransition := time.Now()
			time.Sleep(1 * time.Millisecond)
			got := revision.PlanTransitionTo(targetData)

			want := revision.TransitionPlan{
				TargetData: targetSource,
			}

			testutils.AssertEqual(t, got, want, cmpopts.IgnoreFields(revision.TransitionPlan{}, "UpdatedAt"))

			testutils.AssertTimeAfter(t, got.UpdatedAt, timeBeforeTransition)
			testutils.AssertTimeLocationIs(t, got.UpdatedAt, time.UTC)

			testutils.AssertSlicesIndependent(t, got.TargetData, targetData)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, got, []string{"TargetData", "UpdatedAt"})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil)
		check(testutils.GenerateBytes(t))
	})
}

func TestTransitionSnapshotClone(t *testing.T) {
	t.Run("creates a snapshot copy", func(t *testing.T) {
		check := func(targetData []byte) {
			id := testutils.GenerateID(t)
			scopeID := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)
			variantID := testutils.GenerateID(t)
			baselineData := testutils.GenerateBytes(t)

			createTimeZone := testutils.GenerateTimeZone()
			createdAt := time.Now().In(createTimeZone)

			time.Sleep(1 * time.Millisecond) // ensure updatedAt differs from createdAt, for test data variety
			updateTimeZone := testutils.GenerateTimeZone()
			updatedAt := time.Now().In(updateTimeZone)

			snapshotFromFixture := func() *revision.TransitionSnapshot {
				return &revision.TransitionSnapshot{
					ID:           id,
					ScopeID:      scopeID,
					ResourceID:   resourceID,
					VariantID:    variantID,
					BaselineData: bytes.Clone(baselineData),
					TargetData:   bytes.Clone(targetData),
					CreatedAt:    createdAt,
					UpdatedAt:    updatedAt,
				}
			}

			initial := snapshotFromFixture()

			time.Sleep(1 * time.Millisecond) // ensure setting UpdatedAt to time.Now() in Clone() fails tests
			got := initial.Clone()

			// ensure Clone() did not modify the source snapshot
			testutils.AssertEqual(t, initial, snapshotFromFixture())

			testutils.AssertEqual(t, got, initial)
			testutils.AssertSlicesIndependent(t, got.BaselineData, initial.BaselineData)
			testutils.AssertSlicesIndependent(t, got.TargetData, initial.TargetData)
			testutils.AssertTimeLocationIs(t, got.CreatedAt, createTimeZone)
			testutils.AssertTimeLocationIs(t, got.UpdatedAt, updateTimeZone)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, snapshotFromFixture(), []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
				"UpdatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil)
		check(testutils.GenerateBytes(t))
	})

	t.Run("creates a copy that can be modified independently of the original", func(t *testing.T) {
		check := func(targetData []byte) {
			id := testutils.GenerateID(t)
			scopeID := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)
			variantID := testutils.GenerateID(t)
			baselineData := testutils.GenerateBytes(t)
			createdAt := time.Now()

			time.Sleep(1 * time.Millisecond) // ensure updatedAt differs from createdAt, for test data variety
			updatedAt := time.Now()

			snapshotFromFixture := func() *revision.TransitionSnapshot {
				return &revision.TransitionSnapshot{
					ID:           id,
					ScopeID:      scopeID,
					ResourceID:   resourceID,
					VariantID:    variantID,
					BaselineData: bytes.Clone(baselineData),
					TargetData:   bytes.Clone(targetData),
					CreatedAt:    createdAt,
					UpdatedAt:    updatedAt,
				}
			}

			initial := snapshotFromFixture()
			got := initial.Clone()

			got.ID = testutils.GenerateID(t)
			got.ScopeID = testutils.GenerateID(t)
			got.ResourceID = testutils.GenerateID(t)
			got.VariantID = testutils.GenerateID(t)
			got.BaselineData = testutils.GenerateBytes(t)
			got.TargetData = testutils.GenerateBytes(t)

			time.Sleep(1 * time.Millisecond)
			got.CreatedAt = time.Now()

			time.Sleep(1 * time.Millisecond)
			got.UpdatedAt = time.Now()

			testutils.AssertNotEqual(t, got.ID, initial.ID)
			testutils.AssertNotEqual(t, got.ScopeID, initial.ScopeID)
			testutils.AssertNotEqual(t, got.ResourceID, initial.ResourceID)
			testutils.AssertNotEqual(t, got.VariantID, initial.VariantID)
			testutils.AssertNotEqual(t, got.BaselineData, initial.BaselineData)
			testutils.AssertNotEqual(t, got.TargetData, initial.TargetData)
			testutils.AssertNotEqual(t, got.CreatedAt, initial.CreatedAt)
			testutils.AssertNotEqual(t, got.UpdatedAt, initial.UpdatedAt)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, snapshotFromFixture(), []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "BaselineData", "TargetData", "CreatedAt",
				"UpdatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil)
		check(testutils.GenerateBytes(t))
	})
}
