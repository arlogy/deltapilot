package process

import (
	"context"
	"os"
	"os/signal"
)

// RegisterSignalHandler intercepts signals and invokes a handler when one of them is received.
//   - An empty signal list means all signals.
//   - Signal handling is stopped before the handler is called, restoring the default signal behavior when no
//     other handlers are registered.
//   - When multiple handlers are registered for the same signal, their invocation order is unspecified.
func RegisterSignalHandler(onSignal func(), signals ...os.Signal) context.Context {
	// create and notify ctx when one of the listed signals is received; stop() disables notification
	ctx, stop := signal.NotifyContext(
		context.Background(),
		signals..., // note: empty signal list means all signals
	)

	// wait for signal notification
	go func() {
		<-ctx.Done()
		stop()
		onSignal()
	}()

	return ctx
}
