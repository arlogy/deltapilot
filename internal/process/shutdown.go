package process

import (
	"context"
	"os"
	"syscall"
)

func RegisterShutdownHandler(onShutdown func()) context.Context {
	return RegisterSignalHandler(
		onShutdown,
		os.Interrupt, // SIGINT, e.g. Ctrl+C
		syscall.SIGTERM,
	)
}
