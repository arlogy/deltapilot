package channels

import (
	"errors"
	"fmt"
	"time"
)

// RunWhenUnblocked asynchronously waits until receiving from a channel does not block, then calls a function.
func RunWhenUnblocked(ch <-chan struct{}, onReady func()) {
	go func() {
		<-ch
		onReady()
	}()
}

// WaitForValue waits for a value to be received from a channel. It returns an error on timeout or when the
// channel is closed.
func WaitForValue[T any](timeout time.Duration, ch <-chan T) (T, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case value, ok := <-ch:
		if !ok {
			var zero T
			return zero, errors.New("channel closed before receiving a value")
		}
		return value, nil
	case <-timer.C:
		var zero T
		return zero, fmt.Errorf("timed out after %s waiting for a value", timeout)
	}
}
