package sqldb

import (
	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
)

type TransitionStore struct {
	handle *dbclient.DBHandle
}

func NewTransitionStore(handle *dbclient.DBHandle) *TransitionStore {
	return &TransitionStore{
		handle: handle,
	}
}

func (s *TransitionStore) CanShareState() bool {
	return true
}

func (s *TransitionStore) AddSnapshot(snapshot *revision.TransitionSnapshot) error {
	return createSnapshot(
		s.handle,
		snapshot,
		func(duplicates bool) error {
			switch duplicates {
			case true:
				return storage.WrapSnapshotDuplicateUncategorized(
					snapshot.ID,
					&snapshot.ScopeID,
					snapshot.ResourceID,
					&snapshot.VariantID,
				)
			default:
				return storage.WrapStorageWriteError(
					"failed to create snapshot for ID %q, scope ID %q, resource ID %q, variant ID %q",
					snapshot.ID,
					snapshot.ScopeID,
					snapshot.ResourceID,
					snapshot.VariantID,
				)
			}
		},
	)
}

func (s *TransitionStore) GetByID(id string) (*revision.TransitionSnapshot, error) {
	return getSnapshotByID[revision.TransitionSnapshot](s.handle, id)
}

func (s *TransitionStore) GetByScopeID(scopeID string) ([]*revision.TransitionSnapshot, error) {
	return listSnapshotsByScopeID[revision.TransitionSnapshot](s.handle, &scopeID)
}

func (s *TransitionStore) GetByScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (*revision.TransitionSnapshot, error) {
	return getSnapshotByScopeAndResourceAndVariant[revision.TransitionSnapshot](
		s.handle, &scopeID, resourceID, &variantID,
	)
}

func (s *TransitionStore) MapsID(id string) (bool, error) {
	return hasSnapshotForID[revision.TransitionSnapshot](s.handle, id)
}

func (s *TransitionStore) MapsScopeID(scopeID string) (bool, error) {
	return hasSnapshotForScopeID[revision.TransitionSnapshot](s.handle, &scopeID)
}

func (s *TransitionStore) MapsScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (bool, error) {
	return hasSnapshotForScopeAndResourceAndVariant[revision.TransitionSnapshot](
		s.handle, &scopeID, resourceID, &variantID,
	)
}

func (s *TransitionStore) ApplyTransitionForID(id string, targetData []byte) error {
	plan := revision.PlanTransitionTo(targetData)

	return applySnapshotTransitionForID[revision.TransitionSnapshot](s.handle, id, &plan)
}

func (s *TransitionStore) ApplyTransitionForScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
	targetData []byte,
) error {
	plan := revision.PlanTransitionTo(targetData)

	return applySnapshotTransitionForScopeAndResourceAndVariant[revision.TransitionSnapshot](
		s.handle, scopeID, resourceID, variantID, &plan,
	)
}

func (s *TransitionStore) DeleteByID(id string) (int64, error) {
	return deleteSnapshotsByID[revision.TransitionSnapshot](s.handle, id)
}

func (s *TransitionStore) DeleteByScopeID(scopeID string) (int64, error) {
	return deleteSnapshotsByScopeID[revision.TransitionSnapshot](s.handle, &scopeID)
}

func (s *TransitionStore) DeleteByScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (int64, error) {
	return deleteSnapshotsByScopeAndResourceAndVariant[revision.TransitionSnapshot](
		s.handle, &scopeID, resourceID, &variantID,
	)
}
