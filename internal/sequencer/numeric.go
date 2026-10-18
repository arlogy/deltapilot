package sequencer

import (
	"errors"
	"math"
	"sync"
)

type Numeric struct {
	mu      sync.RWMutex
	counter int64
}

func (n *Numeric) Next() (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.counter == math.MaxInt64 {
		return 0, errors.New("numeric sequencer overflow: maximum value reached")
	}

	n.counter++
	return n.counter, nil
}

func (n *Numeric) GetValue() (int64, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.counter, nil
}

func (n *Numeric) SetValue(v int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.counter = v
	return nil
}
