package testutils_test

import (
	"sync/atomic"
	"testing"

	"github.com/arlogy/deltapilot/internal/testutils"
)

// TestRunConcurrently's outcome depends on the testutils.WorkerCount used by testutils.RunConcurrently().
func TestRunConcurrently(t *testing.T) {
	t.Run("runs a callback function concurrently", func(t *testing.T) {
		flawedCount := func() int64 {
			var result int64
			testutils.RunConcurrently(t, func(workerIdx int) {
				result++ // unguarded modification
			})
			return result
		}()

		exactCount := func() int64 {
			var result int64
			testutils.RunConcurrently(t, func(workerIdx int) {
				atomic.AddInt64(&result, 1) // guarded modification
			})
			return result
		}()

		testutils.AssertEqual(t, flawedCount < testutils.WorkerCount, true)
		testutils.AssertEqual(t, exactCount, int64(testutils.WorkerCount))
	})
}
