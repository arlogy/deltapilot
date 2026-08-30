package memdb

import (
	"sync"

	"github.com/arlogy/deltapilot/internal/ptr"
	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
)

type ChronicleStore struct {
	mu        sync.RWMutex
	snapshots map[string]*revision.ChronicleSnapshot
}

func NewChronicleStore() *ChronicleStore {
	return &ChronicleStore{
		snapshots: make(map[string]*revision.ChronicleSnapshot),
	}
}

func (s *ChronicleStore) CanShareState() bool {
	return false
}

func (s *ChronicleStore) AddSnapshot(snapshot *revision.ChronicleSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if snapshot == nil {
		return storage.WrapSnapshotRequired()
	}

	id := snapshot.ID
	if _, ok := s.snapshots[id]; ok {
		return storage.WrapSnapshotDuplicateID(id)
	}

	s.snapshots[id] = snapshot.Clone()

	return nil
}

func (s *ChronicleStore) GetByID(id string) (*revision.ChronicleSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[id]
	if !ok {
		return nil, storage.WrapSnapshotNotFoundByID(id)
	}

	return snapshot.Clone(), nil
}

func (s *ChronicleStore) GetByScopeID(scopeID *string) ([]*revision.ChronicleSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]*revision.ChronicleSnapshot, 0)
	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) {
			snapshots = append(snapshots, snapshot.Clone())
		}
	}

	return snapshots, nil
}

func (s *ChronicleStore) GetByScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) ([]*revision.ChronicleSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]*revision.ChronicleSnapshot, 0)
	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) &&
			snapshot.ResourceID == resourceID &&
			ptr.EqualString(snapshot.VariantID, variantID) {
			snapshots = append(snapshots, snapshot.Clone())
		}
	}

	return snapshots, nil
}

func (s *ChronicleStore) MapsID(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.snapshots[id]; !ok {
		return false, nil
	}

	return true, nil
}

func (s *ChronicleStore) MapsScopeID(scopeID *string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) {
			return true, nil
		}
	}

	return false, nil
}

func (s *ChronicleStore) MapsScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) &&
			snapshot.ResourceID == resourceID &&
			ptr.EqualString(snapshot.VariantID, variantID) {
			return true, nil
		}
	}

	return false, nil
}

func (s *ChronicleStore) DeleteByID(id string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[id]; !ok {
		return 0, nil
	}

	delete(s.snapshots, id)
	return 1, nil
}

func (s *ChronicleStore) DeleteByScopeID(scopeID *string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) {
			delete(s.snapshots, snapshot.ID)
			deleted++
		}
	}

	return deleted, nil
}

func (s *ChronicleStore) DeleteByScopeAndResourceAndVariant(
	scopeID *string,
	resourceID string,
	variantID *string,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for _, snapshot := range s.snapshots {
		if ptr.EqualString(snapshot.ScopeID, scopeID) &&
			snapshot.ResourceID == resourceID &&
			ptr.EqualString(snapshot.VariantID, variantID) {
			delete(s.snapshots, snapshot.ID)
			deleted++
		}
	}

	return deleted, nil
}
