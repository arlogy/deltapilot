package revision_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/arlogy/deltapilot/internal/ptr"
	"github.com/arlogy/deltapilot/internal/testutils"
	"github.com/arlogy/deltapilot/revision"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewChronicleSnapshot(t *testing.T) {
	t.Run("creates a new snapshot", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			id := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)

			scopeSourceID := ptr.CloneString(scopeID)
			variantSourceID := ptr.CloneString(variantID)
			sourceData := bytes.Clone(data)

			timeBeforeSnapshotCreation := time.Now()
			time.Sleep(1 * time.Millisecond)
			got := revision.NewChronicleSnapshot(id, scopeID, resourceID, variantID, data)

			want := &revision.ChronicleSnapshot{
				ID:         id,
				ScopeID:    scopeSourceID,
				ResourceID: resourceID,
				VariantID:  variantSourceID,
				Data:       sourceData,
			}

			testutils.AssertEqual(t, got, want, cmpopts.IgnoreFields(
				revision.ChronicleSnapshot{}, "CreatedAt",
			))

			testutils.AssertTimeAfter(t, got.CreatedAt, timeBeforeSnapshotCreation)
			testutils.AssertTimeLocationIs(t, got.CreatedAt, time.UTC)

			testutils.AssertScalarPointersIndependent(t, got.ScopeID, scopeID)
			testutils.AssertScalarPointersIndependent(t, got.VariantID, variantID)
			testutils.AssertSlicesIndependent(t, got.Data, data)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, got, []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(nil, nil, testutils.GenerateBytes(t))
		check(nil, testutils.GeneratePointerID(t), nil)
		check(testutils.GeneratePointerID(t), nil, nil)
		check(nil, testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), nil, testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), nil)
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
	})
}

func TestChronicleSnapshotClone(t *testing.T) {
	t.Run("creates a snapshot copy", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			id := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)

			createTimeZone := testutils.GenerateTimeZone()
			createdAt := time.Now().In(createTimeZone)

			snapshotFromFixture := func() *revision.ChronicleSnapshot {
				return &revision.ChronicleSnapshot{
					ID:         id,
					ScopeID:    ptr.CloneString(scopeID),
					ResourceID: resourceID,
					VariantID:  ptr.CloneString(variantID),
					Data:       bytes.Clone(data),
					CreatedAt:  createdAt,
				}
			}

			initial := snapshotFromFixture()

			time.Sleep(1 * time.Millisecond) // ensure setting CreatedAt to time.Now() in Clone() fails tests
			got := initial.Clone()

			// ensure Clone() did not modify the source snapshot
			testutils.AssertEqual(t, initial, snapshotFromFixture())

			testutils.AssertEqual(t, got, initial)
			testutils.AssertScalarPointersIndependent(t, got.ScopeID, initial.ScopeID)
			testutils.AssertScalarPointersIndependent(t, got.VariantID, initial.VariantID)
			testutils.AssertSlicesIndependent(t, got.Data, initial.Data)
			testutils.AssertTimeLocationIs(t, got.CreatedAt, createTimeZone)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, snapshotFromFixture(), []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(nil, nil, testutils.GenerateBytes(t))
		check(nil, testutils.GeneratePointerID(t), nil)
		check(testutils.GeneratePointerID(t), nil, nil)
		check(nil, testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), nil, testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), nil)
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
	})

	t.Run("creates a copy that can be modified independently of the original", func(t *testing.T) {
		check := func(scopeID *string, variantID *string, data []byte) {
			id := testutils.GenerateID(t)
			resourceID := testutils.GenerateID(t)
			createdAt := time.Now()

			snapshotFromFixture := func() *revision.ChronicleSnapshot {
				return &revision.ChronicleSnapshot{
					ID:         id,
					ScopeID:    ptr.CloneString(scopeID),
					ResourceID: resourceID,
					VariantID:  variantID,
					Data:       bytes.Clone(data),
					CreatedAt:  createdAt,
				}
			}

			initial := snapshotFromFixture()
			got := initial.Clone()

			got.ID = testutils.GenerateID(t)
			got.ScopeID = testutils.GeneratePointerID(t)
			got.ResourceID = testutils.GenerateID(t)
			got.VariantID = testutils.GeneratePointerID(t)
			got.Data = testutils.GenerateBytes(t)

			time.Sleep(1 * time.Millisecond)
			got.CreatedAt = time.Now()

			testutils.AssertNotEqual(t, got.ID, initial.ID)
			testutils.AssertNotEqual(t, got.ScopeID, initial.ScopeID)
			testutils.AssertNotEqual(t, got.ResourceID, initial.ResourceID)
			testutils.AssertNotEqual(t, got.VariantID, initial.VariantID)
			testutils.AssertNotEqual(t, got.Data, initial.Data)
			testutils.AssertNotEqual(t, got.CreatedAt, initial.CreatedAt)

			// detect newly added fields so they can be accounted for in tests when necessary
			testutils.AssertFieldNamesEqual(t, snapshotFromFixture(), []string{
				"ID", "ScopeID", "ResourceID", "VariantID", "Data", "CreatedAt",
			})
		}

		// use individual calls instead of a loop so that each case gets isolated arguments
		check(nil, nil, nil)
		check(testutils.GeneratePointerID(t), nil, nil)
		check(nil, testutils.GeneratePointerID(t), nil)
		check(nil, nil, testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), nil)
		check(testutils.GeneratePointerID(t), nil, testutils.GenerateBytes(t))
		check(nil, testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
		check(testutils.GeneratePointerID(t), testutils.GeneratePointerID(t), testutils.GenerateBytes(t))
	})
}
