package transport

import (
	"log"
)

type Logger interface {
	LogInfo(format string, v ...any)
	LogError(format string, v ...any)
}

func NewtDefaultLogger() Logger {
	return &basicLogger{}
}

func NopLogger() Logger {
	return nopLogger
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

var nopLogger = &silentLogger{}
