package revision

import (
	"bytes"
	"time"
)

/*
TransitionSnapshot represents a resource transition from a baseline state to a target state.
The baseline state establishes a reference for comparing the target state.

The following relational schema is provided for clarity.

	transition_snapshot
	    with a unique constraint on (scope_id, resource_id, variant_id)
	+---------+----------+-------------+------------+---------------+-------------+-----+
	| id (PK) | scope_id | resource_id | variant_id | baseline_data | target_data | ... |
	+---------+----------+-------------+------------+---------------+-------------+-----+
	| snap-1  | scope-1  | item-1      | variant-1  | ...           | ...         |     |
	| snap-2  | scope-2  | item-1      | variant-1  | ...           | ...         |     |
	| snap-3  | scope-1  | item-1      | variant-2  | ...           | ...         |     |
	| snap-4  | scope-1  | item-2      | variant-1  | ...           | ...         |     |
	+---------+----------+-------------+------------+---------------+-------------+-----+

ID is a surrogate key that uniquely identifies a snapshot, in addition to its (ScopeID, ResourceID, VariantID)
identity.

ScopeID identifies the scope within which a snapshot is managed.

ResourceID identifies the resource associated with a snapshot.

VariantID distinguishes multiple snapshots of the same resource. Therefore, the tuple (ScopeID, ResourceID,
VariantID) uniquely identifies a snapshot. However, the semantics of VariantID are not defined or enforced by
the model and may vary by use case.
  - E.g. when VariantID values are global identifiers:
    `(ScopeID="X", ResourceID="A", VariantID="user-1")`,
    `(ScopeID="X", ResourceID="B", VariantID="user-1")`;
    `user-1` refers to the same variant across resources; it has global meaning.
  - E.g. when VariantID values are resource-specific identifiers:
    `(ScopeID="X", ResourceID="A", VariantID="default")`,
    `(ScopeID="X", ResourceID="B", VariantID="default")`;
    `default` is interpreted relative to each resource; it has meaning only within a given resource.
*/
type TransitionSnapshot struct {
	ID string

	ScopeID    string
	ResourceID string
	VariantID  string

	BaselineData []byte
	TargetData   []byte

	CreatedAt time.Time
	UpdatedAt time.Time
}

type TransitionPlan struct {
	TargetData []byte
	UpdatedAt  time.Time
}

func NewTransitionSnapshot(
	id string,
	scopeID string,
	resourceID string,
	variantID string,
	baselineData []byte,
	targetData []byte,
) *TransitionSnapshot {
	now := time.Now().UTC()
	return &TransitionSnapshot{
		ID:           id,
		ScopeID:      scopeID,
		ResourceID:   resourceID,
		VariantID:    variantID,
		BaselineData: bytes.Clone(baselineData),
		TargetData:   bytes.Clone(targetData),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// PlanTransitionTo returns the information needed to update a TransitionSnapshot to a new target state while
// preserving its baseline state.
func PlanTransitionTo(targetData []byte) TransitionPlan {
	return TransitionPlan{
		TargetData: bytes.Clone(targetData),
		UpdatedAt:  time.Now().UTC(),
	}
}

func (s *TransitionSnapshot) Clone() *TransitionSnapshot {
	return &TransitionSnapshot{
		ID:           s.ID,
		ScopeID:      s.ScopeID,
		ResourceID:   s.ResourceID,
		VariantID:    s.VariantID,
		BaselineData: bytes.Clone(s.BaselineData),
		TargetData:   bytes.Clone(s.TargetData),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}
