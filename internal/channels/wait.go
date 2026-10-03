package channels

import "time"

// RunOnReceive asynchronously waits for a value from a receiver channel, then calls a function in response.
func RunOnReceive(ch <-chan struct{}, onReceived func()) {
	go func() {
		<-ch
		onReceived()
	}()
}

// WaitForReturn waits until a function returns or timeout occurs.
func WaitForReturn(timeout time.Duration, operation func()) bool {
	done := make(chan struct{})

	go func() {
		operation()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}
