package testutils

import "github.com/arlogy/deltapilot/internal/sequencer"

type StubInfo struct {
	ReturnOverrides    map[string]any
	OverrideSequencers map[string]sequencer.ValueAdvancer[int64]
}
