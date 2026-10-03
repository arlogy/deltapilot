package transport

import (
	"log"
)

// Logger is the logging interface used by the client-server architecture.
//
// If a logger is shared between clients, servers, or other concurrent operations, the caller is responsible
// for ensuring that it is safe for concurrent use. As a result, NewtDefaultLogger() and NewNopLogger() always
// return a new instance.
type Logger interface {
	LogInfo(format string, v ...any)
	LogError(format string, v ...any)
}

func NewtDefaultLogger() Logger {
	return &basicLogger{}
}

func NewNopLogger() Logger {
	return &silentLogger{}
}

type basicLogger struct{}

func (l *basicLogger) LogInfo(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func (l *basicLogger) LogError(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}

type silentLogger struct{}

func (l *silentLogger) LogInfo(format string, v ...any) {}

func (l *silentLogger) LogError(format string, v ...any) {}
