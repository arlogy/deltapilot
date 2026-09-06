package sqldb

import (
	"time"

	"github.com/arlogy/deltapilot/revision"
)

// transitionRow contains GORM tags for controlling GORM's automatic behavior.
//   - We use it only for certain database write operations on revision.TransitionSnapshot.
//   - We do not test its effects in our test suite, as this behavior is specific to GORM and testing it would
//     require distinguishing between zero non-zero time.Time values.
type transitionRow struct {
	ID           string
	ScopeID      string
	ResourceID   string
	VariantID    string
	BaselineData []byte
	TargetData   []byte
	CreatedAt    time.Time `gorm:"autoCreateTime:false"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime:false"`
}

func newTransitionRow(snapshot *revision.TransitionSnapshot) *transitionRow {
	if snapshot == nil {
		return nil
	}

	return &transitionRow{
		ID:           snapshot.ID,
		ScopeID:      snapshot.ScopeID,
		ResourceID:   snapshot.ResourceID,
		VariantID:    snapshot.VariantID,
		BaselineData: snapshot.BaselineData,
		TargetData:   snapshot.TargetData,
		CreatedAt:    snapshot.CreatedAt,
		UpdatedAt:    snapshot.UpdatedAt,
	}
}

// https://gorm.io/docs/conventions.html
func (r transitionRow) TableName() string {
	return "transition_snapshot"
}
