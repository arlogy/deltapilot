package testutils

import (
	"sync"
	"testing"
)

// WorkerCount is tuned so that TestRunConcurrently() passes often enough in local test environments.
// A passing test is a good indication that RunConcurrently() is reliable.
// Higher values increase test execution time.
const WorkerCount = 100

func RunConcurrently(t *testing.T, fn func(workerIdx int)) {
	startGate := make(chan struct{})

	var wg sync.WaitGroup
	for i := range WorkerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startGate // wait for all goroutines to be ready
			fn(i)
		}()
	}

	close(startGate) // release all waiting goroutines; they start concurrently
	wg.Wait()
}
