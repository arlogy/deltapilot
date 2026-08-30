package storage

import (
	"github.com/arlogy/deltapilot/revision"
)

type ChronicleStore interface {
	// --------
	// Non-CRUD
	// --------

	// CanShareState indicates whether multiple instances of the store implementation can share the same
	// underlying state, such as a common database.
	//   - Must return false when separate instances never share state.
	//   - Must return true when separate instances can share the same underlying state; so creating another
	//     instance does not imply isolation.
	CanShareState() bool

	// ------
	// Create
	// ------
	//   - input pointers can be safely modified once the operation completes; changes will not affect the
	//     store
	//   - input datetimes, whether standalone values or fields, preserve their represented instant, but their
	//     time zones may not be preserved

	AddSnapshot(snapshot *revision.ChronicleSnapshot) error

	// ----
	// Read
	// ----
	//   - when multiple records are returned, their order is unspecified
	//   - returned pointers are safe to modify; changes will not affect the store
	//   - retrieved datetime values preserve their represented instant, but may not preserve their original
	//     time zones

	GetByID(id string) (*revision.ChronicleSnapshot, error)
	GetByScopeID(scopeID *string) ([]*revision.ChronicleSnapshot, error)
	GetByScopeAndResourceAndVariant(
		scopeID *string,
		resourceID string,
		variantID *string,
	) ([]*revision.ChronicleSnapshot, error)

	MapsID(id string) (bool, error)
	MapsScopeID(scopeID *string) (bool, error)
	MapsScopeAndResourceAndVariant(scopeID *string, resourceID string, variantID *string) (bool, error)

	// ------
	// Delete
	// ------

	DeleteByID(id string) (int64, error)
	DeleteByScopeID(scopeID *string) (int64, error)
	DeleteByScopeAndResourceAndVariant(scopeID *string, resourceID string, variantID *string) (int64, error)
}
