package storage

import (
	"github.com/arlogy/deltapilot/revision"
)

type TransitionStore interface {
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

	AddSnapshot(snapshot *revision.TransitionSnapshot) error

	// ----
	// Read
	// ----
	//   - when multiple records are returned, their order is unspecified
	//   - returned pointers are safe to modify; changes will not affect the store
	//   - retrieved datetime values preserve their represented instant, but may not preserve their original
	//     time zones

	GetByID(id string) (*revision.TransitionSnapshot, error)
	GetByScopeID(scopeID string) ([]*revision.TransitionSnapshot, error)
	GetByScopeAndResourceAndVariant(
		scopeID string,
		resourceID string,
		variantID string,
	) (*revision.TransitionSnapshot, error)

	MapsID(id string) (bool, error)
	MapsScopeID(scopeID string) (bool, error)
	MapsScopeAndResourceAndVariant(scopeID string, resourceID string, variantID string) (bool, error)

	// ------
	// Update
	// ------

	ApplyTransitionForID(id string, targetData []byte) error
	ApplyTransitionForScopeAndResourceAndVariant(
		scopeID string,
		resourceID string,
		variantID string,
		targetData []byte,
	) error

	// ------
	// Delete
	// ------

	DeleteByID(id string) (int64, error)
	DeleteByScopeID(scopeID string) (int64, error)
	DeleteByScopeAndResourceAndVariant(scopeID string, resourceID string, variantID string) (int64, error)
}

// UpsertTransitionSnapshot updates an existing snapshot or creates a new one if none exists.
// The baseline data is used only when creating a new snapshot.
func UpsertTransitionSnapshot(
	store TransitionStore,
	scopeID string,
	resourceID string,
	variantID string,
	baselineData []byte,
	targetData []byte,
	generateID func() string,
) error {
	// this algorithm does not require mutual exclusion locks because each store operation invoked below is
	// expected to execute concurrently without causing inconsistencies in the store

	snapshotExists, err := store.MapsScopeAndResourceAndVariant(scopeID, resourceID, variantID)
	if err != nil {
		return err
	}

	if snapshotExists {
		return store.ApplyTransitionForScopeAndResourceAndVariant(scopeID, resourceID, variantID, targetData)
	}

	id := generateID()
	snapshot := revision.NewTransitionSnapshot(id, scopeID, resourceID, variantID, baselineData, targetData)
	return store.AddSnapshot(snapshot)
}
