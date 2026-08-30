package storage

import (
	"errors"
	"fmt"

	"github.com/arlogy/deltapilot/internal/ptr"
)

var (
	ErrSnapshotRequired  = errors.New("snapshot is required")
	ErrSnapshotDuplicate = errors.New("snapshot already exists")
	ErrSnapshotRetrieval = errors.New("snapshot cannot be retrieved")

	ErrStorageRead  = errors.New("storage read error")
	ErrStorageWrite = errors.New("storage write error")
)

func WrapSnapshotDuplicateID(id string) error {
	return fmt.Errorf("%w: ID %q", ErrSnapshotDuplicate, id)
}

func WrapSnapshotDuplicateScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) error {
	return fmt.Errorf(
		"%w: scope ID %q, resource ID %q, variant ID %q",
		ErrSnapshotDuplicate,
		scopeID,
		resourceID,
		variantID,
	)
}

func WrapSnapshotDuplicateUncategorized(
	id string,
	scopeID *string,
	resourceID string,
	variantID *string,
) error {
	return fmt.Errorf(
		"%w: ID %q alone, or scope ID %s, resource ID %q, variant ID %s combination",
		ErrSnapshotDuplicate,
		id,
		ptr.StringQuotedOrNull(scopeID),
		resourceID,
		ptr.StringQuotedOrNull(variantID),
	)
}

func WrapSnapshotNotFoundByID(id string) error {
	return fmt.Errorf("%w: ID %q", ErrSnapshotRetrieval, id)
}

func WrapSnapshotNotFoundByScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) error {
	return fmt.Errorf(
		"%w: scope ID %s, resource ID %q, variant ID %s",
		ErrSnapshotRetrieval,
		ptr.StringQuotedOrNull(scopeID),
		resourceID,
		ptr.StringQuotedOrNull(variantID),
	)
}

func WrapSnapshotRequired() error {
	return fmt.Errorf("%w", ErrSnapshotRequired)
}

func WrapStorageReadError(detailFormat string, formatArgs ...any) error {
	detailMsg := fmt.Sprintf(detailFormat, formatArgs...)
	return fmt.Errorf("%w: %s", ErrStorageRead, detailMsg)
}

func WrapStorageWriteError(detailFormat string, formatArgs ...any) error {
	detailMsg := fmt.Sprintf(detailFormat, formatArgs...)
	return fmt.Errorf("%w: %s", ErrStorageWrite, detailMsg)
}
