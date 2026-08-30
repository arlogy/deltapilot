package failures

import (
	"errors"
	"fmt"
)

// AsErrorWithSemantics converts a regular error into another regular error that reflects any additional
// information carried by the original error. Errors without such information are returned unchanged.
func AsErrorWithSemantics(err error) error {
	if err, ok := errors.AsType[*DetailedError](err); ok {
		return fmt.Errorf(
			"a detailed error occurred:\n- public cause: %w\n- internal cause: %v",
			err.PublicCause,
			err.InternalCause,
		)
	}
	return err
}
