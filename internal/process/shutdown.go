package process

import (
	"context"
	"os"
	"syscall"
)

// RegisterShutdownHandler registers a handler for shutdown signals using RegisterSignalHandler().
func RegisterShutdownHandler(onShutdown func()) context.Context {
	return RegisterSignalHandler(
		onShutdown,
		os.Interrupt, // SIGINT, e.g. Ctrl+C
		syscall.SIGTERM,
	)
}
