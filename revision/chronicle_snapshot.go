package revision

import (
	"bytes"
	"time"

	"github.com/arlogy/deltapilot/internal/ptr"
)

/*
ChronicleSnapshot represents a resource state at a specific point in time.
Ultimately, these snapshots form a chronological version history for each resource.

The following relational schema is provided for clarity.

	chronicle_snapshot
	+---------+----------+-------------+------------+------+-----+
	| id (PK) | scope_id | resource_id | variant_id | data | ... |
	+---------+----------+-------------+------------+------+-----+
	| snap-1  | ...      | item-1      | ...        | ...  |     |
	| snap-2  | ...      | item-1      | ...        | ...  |     |
	| snap-3  | ...      | item-2      | ...        | ...  |     |
	| snap-4  | ...      | item-1      | ...        | ...  |     |
	+---------+----------+-------------+------------+------+-----+

ID is a surrogate key that uniquely identifies a snapshot, and it is the only snapshot identifier.

ResourceID identifies the resource associated with a snapshot.

ScopeID optionnaly identifies the scope within which a snapshot is managed. VariantID optionnaly identifies
which variant of a resource a snapshot captures. They are meant to enable a more granular correlation across
snapshot types.
*/
type ChronicleSnapshot struct {
	ID string

	ScopeID    *string
	ResourceID string
	VariantID  *string

	Data []byte

	CreatedAt time.Time
}

func NewChronicleSnapshot(
	id string,
	scopeID *string,
	resourceID string,
	variantID *string,
	data []byte,
) *ChronicleSnapshot {
	return &ChronicleSnapshot{
		ID:         id,
		ScopeID:    ptr.CloneString(scopeID),
		ResourceID: resourceID,
		VariantID:  ptr.CloneString(variantID),
		Data:       bytes.Clone(data),
		CreatedAt:  time.Now().UTC(),
	}
}

func (s *ChronicleSnapshot) Clone() *ChronicleSnapshot {
	return &ChronicleSnapshot{
		ID:         s.ID,
		ScopeID:    ptr.CloneString(s.ScopeID),
		ResourceID: s.ResourceID,
		VariantID:  ptr.CloneString(s.VariantID),
		Data:       bytes.Clone(s.Data),
		CreatedAt:  s.CreatedAt,
	}
}
