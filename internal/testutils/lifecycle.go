package testutils

import (
	"context"
	"fmt"
	"time"
)

func StopOperation(
	name string,
	ctxCancel context.CancelFunc,
	stoppedCh <-chan struct{},
	timeout time.Duration,
) error {
	ctxCancel() // cancel the context associated with the operation

	select {
	case <-stoppedCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %s waiting for %s to stop", timeout, name)
	}
}
