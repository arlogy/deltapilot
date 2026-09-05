package memdb

import (
	"sync"

	"github.com/arlogy/deltapilot/revision"
	"github.com/arlogy/deltapilot/storage"
)

type TransitionStore struct {
	mu        sync.RWMutex
	snapshots map[string]*revision.TransitionSnapshot
}

func NewTransitionStore() *TransitionStore {
	return &TransitionStore{
		snapshots: make(map[string]*revision.TransitionSnapshot),
	}
}

func (s *TransitionStore) CanShareState() bool {
	return false
}

func (s *TransitionStore) AddSnapshot(snapshot *revision.TransitionSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if snapshot == nil {
		return storage.WrapSnapshotRequired()
	}

	id := snapshot.ID
	if _, ok := s.snapshots[id]; ok {
		return storage.WrapSnapshotDuplicateID(id)
	}

	// we call hasScopeAndResourceAndVariant() instead of MapsScopeAndResourceAndVariant() to avoid a
	// deadlock, since s.mu is already write-locked
	scopeID, resourceID, variantID := snapshot.ScopeID, snapshot.ResourceID, snapshot.VariantID
	snapshotExists := s.hasScopeAndResourceAndVariant(scopeID, resourceID, variantID)
	if snapshotExists {
		return storage.WrapSnapshotDuplicateScopeAndResourceAndVariant(scopeID, resourceID, variantID)
	}

	s.snapshots[id] = snapshot.Clone()

	return nil
}

func (s *TransitionStore) GetByID(id string) (*revision.TransitionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[id]
	if !ok {
		return nil, storage.WrapSnapshotNotFoundByID(id)
	}

	return snapshot.Clone(), nil
}

func (s *TransitionStore) GetByScopeID(scopeID string) ([]*revision.TransitionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]*revision.TransitionSnapshot, 0)
	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID {
			snapshots = append(snapshots, snapshot.Clone())
		}
	}

	return snapshots, nil
}

func (s *TransitionStore) GetByScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (*revision.TransitionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findByScopeAndResourceAndVariant(scopeID, resourceID, variantID, true)
}

func (s *TransitionStore) MapsID(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.snapshots[id]; !ok {
		return false, nil
	}

	return true, nil
}

func (s *TransitionStore) MapsScopeID(scopeID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID {
			return true, nil
		}
	}

	return false, nil
}

func (s *TransitionStore) MapsScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.hasScopeAndResourceAndVariant(scopeID, resourceID, variantID), nil
}

func (s *TransitionStore) ApplyTransitionForID(id string, targetData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, ok := s.snapshots[id]
	if !ok {
		return storage.WrapSnapshotNotFoundByID(id)
	}

	plan := revision.PlanTransitionTo(targetData)
	snapshot.TargetData = plan.TargetData
	snapshot.UpdatedAt = plan.UpdatedAt

	return nil
}

func (s *TransitionStore) ApplyTransitionForScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
	targetData []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// we call findByScopeAndResourceAndVariant() instead of GetByScopeAndResourceAndVariant() to avoid a
	// deadlock, since s.mu is already write-locked
	snapshot, err := s.findByScopeAndResourceAndVariant(scopeID, resourceID, variantID, false)
	if err != nil {
		return err
	}

	plan := revision.PlanTransitionTo(targetData)
	snapshot.TargetData = plan.TargetData
	snapshot.UpdatedAt = plan.UpdatedAt

	return nil
}

func (s *TransitionStore) DeleteByID(id string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[id]; !ok {
		return 0, nil
	}

	delete(s.snapshots, id)
	return 1, nil
}

func (s *TransitionStore) DeleteByScopeID(scopeID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID {
			delete(s.snapshots, snapshot.ID)
			deleted++
		}
	}

	return deleted, nil
}

func (s *TransitionStore) DeleteByScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID &&
			snapshot.ResourceID == resourceID &&
			snapshot.VariantID == variantID {
			delete(s.snapshots, snapshot.ID)
			deleted++
		}
	}

	return deleted, nil
}

func (s *TransitionStore) findByScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
	cloneMatch bool,
) (*revision.TransitionSnapshot, error) {
	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID &&
			snapshot.ResourceID == resourceID &&
			snapshot.VariantID == variantID {
			if cloneMatch {
				return snapshot.Clone(), nil
			}
			return snapshot, nil
		}
	}

	return nil, storage.WrapSnapshotNotFoundByScopeAndResourceAndVariant(&scopeID, resourceID, &variantID)
}

func (s *TransitionStore) hasScopeAndResourceAndVariant(
	scopeID string,
	resourceID string,
	variantID string,
) bool {
	for _, snapshot := range s.snapshots {
		if snapshot.ScopeID == scopeID &&
			snapshot.ResourceID == resourceID &&
			snapshot.VariantID == variantID {
			return true
		}
	}

	return false
}
