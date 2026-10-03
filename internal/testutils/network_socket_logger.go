package testutils

import "fmt"

// NetSpyLogger is a logger that acts as a spy.
type NetSpyLogger struct {
	Logs []string
}

func (l *NetSpyLogger) LogInfo(format string, v ...any) {
	l.Logs = append(l.Logs, "[INFO] "+fmt.Sprintf(format, v...))
}

func (l *NetSpyLogger) LogError(format string, v ...any) {
	l.Logs = append(l.Logs, "[ERROR] "+fmt.Sprintf(format, v...))
}
