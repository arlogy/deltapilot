package sequencer

import (
	"errors"
	"math"
	"sync/atomic"
)

type Numeric struct {
	counter atomic.Int64
}

func (n *Numeric) Next() (int64, error) {
	if n.counter.Load() == math.MaxInt64 {
		return 0, errors.New("numeric sequencer overflow: maximum value reached")
	}

	return n.counter.Add(1), nil
}

func (n *Numeric) GetCurrent() (int64, error) {
	return n.counter.Load(), nil
}

func (n *Numeric) SetCurrent(v int64) error {
	n.counter.Store(v)
	return nil
}
