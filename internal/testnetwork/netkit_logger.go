package testnetwork

import "fmt"

func NetFormatInfo(format string, v ...any) string {
	return "[INFO] " + fmt.Sprintf(format, v...)
}

func NetFormatError(format string, v ...any) string {
	return "[ERROR] " + fmt.Sprintf(format, v...)
}

// NetSpyLogger is a logger that acts as a spy.
type NetSpyLogger struct {
	Logs []string
}

func (l *NetSpyLogger) LogInfo(format string, v ...any) {
	l.Logs = append(l.Logs, NetFormatInfo(format, v...))
}

func (l *NetSpyLogger) LogError(format string, v ...any) {
	l.Logs = append(l.Logs, NetFormatError(format, v...))
}

func (l *NetSpyLogger) Reset() {
	l.Logs = []string{}
}
