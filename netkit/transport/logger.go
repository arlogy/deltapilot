package transport

import (
	"log"
)

type NetLogger interface {
	LogInfo(format string, v ...any)
	LogError(format string, v ...any)
}

func NewNetLogger() NetLogger {
	return &defaultLogger{}
}

type defaultLogger struct{}

func (l *defaultLogger) LogInfo(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func (l *defaultLogger) LogError(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}
