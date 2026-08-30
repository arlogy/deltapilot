package sqldb

import (
	"github.com/arlogy/deltapilot/internal/dbclient"
	"github.com/arlogy/deltapilot/internal/ptr"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
)

type ChronicleStore struct {
	handle *dbclient.DBHandle
}

func NewChronicleStore(handle *dbclient.DBHandle) *ChronicleStore {
	return &ChronicleStore{
		handle: handle,
	}
}

func (s *ChronicleStore) CanShareState() bool {
	return true
}

func (s *ChronicleStore) AddSnapshot(snapshot *revision.ChronicleSnapshot) error {
	return createSnapshot(
		s.handle,
		newChronicleRow(snapshot),
		func(duplicates bool) error {
			switch duplicates {
			case true:
				return storage.WrapSnapshotDuplicateUncategorized(
					snapshot.ID,
					snapshot.ScopeID,
					snapshot.ResourceID,
					snapshot.VariantID,
				)
			default:
				return storage.WrapStorageWriteError(
					"failed to create snapshot for ID %q, scope ID %s, resource ID %q, variant ID %s",
					snapshot.ID,
					ptr.StringQuotedOrNull(snapshot.ScopeID),
					snapshot.ResourceID,
					ptr.StringQuotedOrNull(snapshot.VariantID),
				)
			}
		},
	)
}

func (s *ChronicleStore) GetByID(id string) (*revision.ChronicleSnapshot, error) {
	return getSnapshotByID[revision.ChronicleSnapshot](s.handle, id)
}

func (s *ChronicleStore) GetByScopeID(scopeID *string) ([]*revision.ChronicleSnapshot, error) {
	return listSnapshotsByScopeID[revision.ChronicleSnapshot](s.handle, scopeID)
}

func (s *ChronicleStore) GetByScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) ([]*revision.ChronicleSnapshot, error) {
	return listSnapshotsByScopeAndResourceAndVariant[revision.ChronicleSnapshot](
		s.handle, scopeID, resourceID, variantID,
	)
}

func (s *ChronicleStore) MapsID(id string) (bool, error) {
	return hasSnapshotForID[revision.ChronicleSnapshot](s.handle, id)
}

func (s *ChronicleStore) MapsScopeID(scopeID *string) (bool, error) {
	return hasSnapshotForScopeID[revision.ChronicleSnapshot](s.handle, scopeID)
}

func (s *ChronicleStore) MapsScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) (bool, error) {
	return hasSnapshotForScopeAndResourceAndVariant[revision.ChronicleSnapshot](
		s.handle, scopeID, resourceID, variantID,
	)
}

func (s *ChronicleStore) DeleteByID(id string) (int64, error) {
	return deleteSnapshotsByID[revision.ChronicleSnapshot](s.handle, id)
}

func (s *ChronicleStore) DeleteByScopeID(scopeID *string) (int64, error) {
	return deleteSnapshotsByScopeID[revision.ChronicleSnapshot](s.handle, scopeID)
}

func (s *ChronicleStore) DeleteByScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) (int64, error) {
	return deleteSnapshotsByScopeAndResourceAndVariant[revision.ChronicleSnapshot](
		s.handle, scopeID, resourceID, variantID,
	)
}
