package sqldb

import (
	"time"

	"github.com/arlogy/deltapilot/revision"
)

// chronicleRow contains metadata used to customize GORM behavior for certain database operations on
// revision.ChronicleSnapshot.
type chronicleRow struct {
	ID         string
	ScopeID    *string
	ResourceID string
	VariantID  *string
	Data       []byte
	CreatedAt  time.Time `gorm:"not null;autoCreateTime:false"`
}

func newChronicleRow(snapshot *revision.ChronicleSnapshot) *chronicleRow {
	if snapshot == nil {
		return nil
	}

	return &chronicleRow{
		ID:         snapshot.ID,
		ScopeID:    snapshot.ScopeID,
		ResourceID: snapshot.ResourceID,
		VariantID:  snapshot.VariantID,
		Data:       snapshot.Data,
		CreatedAt:  snapshot.CreatedAt,
	}
}

// https://gorm.io/docs/conventions.html
func (r chronicleRow) TableName() string {
	return "chronicle_snapshot"
}
