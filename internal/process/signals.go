package process

import (
	"context"
	"os"
	"os/signal"
)

// CreateContextFromSignals creates a context that is notified when one of the listed signals is intercepted,
// or any signal when none is listed.
//   - <-ctx.Done() can be used to wait for the first signal notification.
//   - stop() can be called to stop receiving signal notifications, restoring the default signal behavior for
//     any of these signals that no context is notified of anymore.
//   - When multiple contexts are created to receive notification of the same signals, the notification order
//     is unspecified.
func CreateContextFromSignals(signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	return ctx, stop
}
