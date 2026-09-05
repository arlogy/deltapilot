package sqldb

import (
	"errors"

	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/internal/failures"
	"github.com/arlogy/deltapilot/internal/ptr"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
	"gorm.io/gorm"
)

func createSnapshot[T any](
	handle *dbclient.DBHandle,
	snapshot *T,
	publicErr func(duplicates bool) error,
) error {
	if snapshot == nil {
		return storage.WrapSnapshotRequired()
	}

	privateErr := handle.DB.Create(snapshot).Error
	if errors.Is(privateErr, gorm.ErrDuplicatedKey) {
		return &failures.DetailedError{
			PublicCause:   publicErr(true),
			InternalCause: privateErr,
		}
	}
	if privateErr != nil {
		return &failures.DetailedError{
			PublicCause:   publicErr(false),
			InternalCause: privateErr,
		}
	}

	return nil
}

func getSnapshotByID[T any](handle *dbclient.DBHandle, id string) (*T, error) {
	var snapshot T

	err := handle.DB.Where("id = ?", id).First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, storage.WrapSnapshotNotFoundByID(id)
	}
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause:   storage.WrapStorageReadError("failed to get snapshot for ID %q", id),
			InternalCause: err,
		}
	}

	return &snapshot, nil
}

func getSnapshotByScopeAndResourceAndVariant[T any](
	handle *dbclient.DBHandle,
	scopeID *string,
	resourceID string,
	variantID *string,
) (*T, error) {
	var snapshot T

	db := handle.DB
	if scopeID == nil && variantID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id IS NULL", resourceID)
	} else if scopeID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id = ?", resourceID, *variantID)
	} else if variantID == nil {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id IS NULL", *scopeID, resourceID)
	} else {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id = ?", *scopeID, resourceID, *variantID)
	}

	err := db.First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(scopeID, resourceID, variantID)
	}
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause: storage.WrapStorageReadError(
				"failed to get snapshot for scope ID %s, resource ID %q, variant ID %s",
				ptr.StringQuotedOrNull(scopeID),
				resourceID,
				ptr.StringQuotedOrNull(variantID),
			),
			InternalCause: err,
		}
	}

	return &snapshot, nil
}

func listSnapshotsByScopeID[T any](handle *dbclient.DBHandle, scopeID *string) ([]*T, error) {
	var snapshots []*T

	db := handle.DB
	if scopeID == nil {
		db = db.Where("scope_id IS NULL")
	} else {
		db = db.Where("scope_id = ?", *scopeID)
	}

	err := db.Find(&snapshots).Error
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause: storage.WrapStorageReadError(
				"failed to list snapshots for scope ID %s",
				ptr.StringQuotedOrNull(scopeID),
			),
			InternalCause: err,
		}
	}

	return snapshots, nil
}

func listSnapshotsByScopeAndResourceAndVariant[T any](
	handle *dbclient.DBHandle,
	scopeID *string,
	resourceID string,
	variantID *string,
) ([]*T, error) {
	var snapshots []*T

	db := handle.DB
	if scopeID == nil && variantID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id IS NULL", resourceID)
	} else if scopeID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id = ?", resourceID, *variantID)
	} else if variantID == nil {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id IS NULL", *scopeID, resourceID)
	} else {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id = ?", *scopeID, resourceID, *variantID)
	}

	err := db.Find(&snapshots).Error
	if err != nil {
		return nil, &failures.DetailedError{
			PublicCause: storage.WrapStorageReadError(
				"failed to list snapshots for scope ID %s, resource ID %q, variant ID %s",
				ptr.StringQuotedOrNull(scopeID),
				resourceID,
				ptr.StringQuotedOrNull(variantID),
			),
			InternalCause: err,
		}
	}

	return snapshots, nil
}

func hasSnapshotForID[T any](handle *dbclient.DBHandle, id string) (bool, error) {
	var snapshot T

	db := handle.DB
	err := db.Where("id = ?", id).Select("id").First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, &failures.DetailedError{
			PublicCause:   storage.WrapStorageReadError("failed to check snapshot existence for ID %q", id),
			InternalCause: err,
		}
	}

	return true, nil
}

func hasSnapshotForScopeID[T any](handle *dbclient.DBHandle, scopeID *string) (bool, error) {
	var snapshot T

	db := handle.DB
	if scopeID == nil {
		db = db.Where("scope_id IS NULL")
	} else {
		db = db.Where("scope_id = ?", *scopeID)
	}

	err := db.Select("id").First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, &failures.DetailedError{
			PublicCause: storage.WrapStorageReadError(
				"failed to check snapshot existence for scope ID %s",
				ptr.StringQuotedOrNull(scopeID),
			),
			InternalCause: err,
		}
	}

	return true, nil
}

func hasSnapshotForScopeAndResourceAndVariant[T any](
	handle *dbclient.DBHandle,
	scopeID *string,
	resourceID string,
	variantID *string,
) (bool, error) {
	var snapshot T

	db := handle.DB
	if scopeID == nil && variantID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id IS NULL", resourceID)
	} else if scopeID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id = ?", resourceID, *variantID)
	} else if variantID == nil {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id IS NULL", *scopeID, resourceID)
	} else {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id = ?", *scopeID, resourceID, *variantID)
	}

	err := db.Select("id").First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, &failures.DetailedError{
			PublicCause: storage.WrapStorageReadError(
				"failed to check snapshot existence for scope ID %s, resource ID %q, variant ID %s",
				ptr.StringQuotedOrNull(scopeID),
				resourceID,
				ptr.StringQuotedOrNull(variantID),
			),
			InternalCause: err,
		}
	}

	return true, nil
}

func applySnapshotTransitionForID[T any](
	handle *dbclient.DBHandle,
	id string,
	plan *revision.TransitionPlan,
) error {
	var snapshot T

	result := handle.DB.
		Model(&snapshot).
		Where("id = ?", id).
		Updates(map[string]any{
			"target_data": plan.TargetData,
			"updated_at":  plan.UpdatedAt,
		})
	if result.Error != nil {
		return &failures.DetailedError{
			PublicCause: storage.WrapStorageWriteError(
				"failed to apply snapshot transition for ID %q",
				id,
			),
			InternalCause: result.Error,
		}
	}

	if result.RowsAffected == 0 {
		return storage.WrapSnapshotNotFoundByID(id)
	}

	// defensive check: this should never occur, as a snapshot is uniquely identified by an ID
	if result.RowsAffected != 1 {
		return storage.WrapStorageWriteError(
			"expected exactly 1 snapshot to be updated for ID %q; updated %d",
			id,
			result.RowsAffected,
		)
	}

	return nil
}

func applySnapshotTransitionForScopeAndResourceAndVariant[T any](
	handle *dbclient.DBHandle,
	scopeID string,
	resourceID string,
	variantID string,
	plan *revision.TransitionPlan,
) error {
	var snapshot T

	result := handle.DB.
		Model(&snapshot).
		Where("scope_id = ? AND resource_id = ? AND variant_id = ?", scopeID, resourceID, variantID).
		Updates(map[string]any{
			"target_data": plan.TargetData,
			"updated_at":  plan.UpdatedAt,
		})
	if result.Error != nil {
		return &failures.DetailedError{
			PublicCause: storage.WrapStorageWriteError(
				"failed to apply snapshot transition for scope ID %q, resource ID %q, variant ID %q",
				scopeID,
				resourceID,
				variantID,
			),
			InternalCause: result.Error,
		}
	}

	if result.RowsAffected == 0 {
		return storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(&scopeID, resourceID, &variantID)
	}

	// defensive check: this should never occur, as a snapshot is uniquely identified by the identifiers
	if result.RowsAffected != 1 {
		return storage.WrapStorageWriteError(
			"expected exactly 1 snapshot to be updated for scope ID %q, resource ID %q, variant ID %q;"+
				" updated %d",
			scopeID,
			resourceID,
			variantID,
			result.RowsAffected,
		)
	}

	return nil
}

func deleteSnapshotsByID[T any](handle *dbclient.DBHandle, id string) (int64, error) {
	var snapshot T

	result := handle.DB.Where("id = ?", id).Delete(&snapshot)
	if result.Error != nil {
		return result.RowsAffected, &failures.DetailedError{
			PublicCause:   storage.WrapStorageWriteError("failed to delete snapshot for ID %q", id),
			InternalCause: result.Error,
		}
	}

	return result.RowsAffected, nil
}

func deleteSnapshotsByScopeID[T any](handle *dbclient.DBHandle, scopeID *string) (int64, error) {
	var snapshot T

	db := handle.DB
	if scopeID == nil {
		db = db.Where("scope_id IS NULL")
	} else {
		db = db.Where("scope_id = ?", *scopeID)
	}

	result := db.Delete(&snapshot)
	if result.Error != nil {
		return result.RowsAffected, &failures.DetailedError{
			PublicCause: storage.WrapStorageWriteError(
				"failed to delete snapshots for scopeID ID %s",
				ptr.StringQuotedOrNull(scopeID),
			),
			InternalCause: result.Error,
		}
	}

	return result.RowsAffected, nil
}

func deleteSnapshotsByScopeAndResourceAndVariant[T any](
	handle *dbclient.DBHandle,
	scopeID *string,
	resourceID string,
	variantID *string,
) (int64, error) {
	var snapshot T

	db := handle.DB
	if scopeID == nil && variantID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id IS NULL", resourceID)
	} else if scopeID == nil {
		db = db.Where("scope_id IS NULL AND resource_id = ? AND variant_id = ?", resourceID, *variantID)
	} else if variantID == nil {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id IS NULL", *scopeID, resourceID)
	} else {
		db = db.Where("scope_id = ? AND resource_id = ? AND variant_id = ?", *scopeID, resourceID, *variantID)
	}

	result := db.Delete(&snapshot)
	if result.Error != nil {
		return result.RowsAffected, &failures.DetailedError{
			PublicCause: storage.WrapStorageWriteError(
				"failed to delete snapshots for scope ID %s, resource ID %q, variant ID %s",
				ptr.StringQuotedOrNull(scopeID),
				resourceID,
				ptr.StringQuotedOrNull(variantID),
			),
			InternalCause: result.Error,
		}
	}

	return result.RowsAffected, nil
}
