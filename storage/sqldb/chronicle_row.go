package sqldb

import (
	"time"

	"github.com/arlogy/deltapilot/revision"
)

// chronicleRow contains GORM tags for controlling GORM's automatic behavior.
//   - We use it only for certain database write operations on revision.ChronicleSnapshot.
//   - We do not test its effects in our test suite, as this behavior is specific to GORM and testing it would
//     require distinguishing between zero non-zero time.Time values.
type chronicleRow struct {
	ID         string
	ScopeID    *string
	ResourceID string
	VariantID  *string
	Data       []byte
	CreatedAt  time.Time `gorm:"autoCreateTime:false"`
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
