package process

import (
	"context"
	"os"
	"os/signal"
)

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
