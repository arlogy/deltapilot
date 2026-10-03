package process

import (
	"context"
	"os"
	"syscall"
)

func CreateShutdownContext() (context.Context, context.CancelFunc) {
	return CreateContextFromSignals(
		os.Interrupt, // SIGINT, e.g. Ctrl+C
		syscall.SIGTERM,
	)
}
