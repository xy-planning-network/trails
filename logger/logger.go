package logger

import "log"

// The Logger interface defines the levels a logging can occur at.
type Logger interface {
	Debug(msg string, data map[string]interface{})
	Error(msg string, data map[string]interface{})
	Fatal(msg string, data map[string]interface{})
	Info(msg string, data map[string]interface{})
	Warn(msg string, data map[string]interface{})
}

const (
	colorRed    = "\x1b[31;1m"
	colorBlue   = "\x1b[34;1m"
	colorYellow = "\x1b[33;1m"
	colorPink   = "\x1b[35;1m"
	colorWhite  = "\x1b[37;1m"
	colorClose  = "\x1b[0m"

	logLevelDebug = "[DEBUG]"
	logLevelInfo  = "[INFO]"
	logLevelWarn  = "[WARN]"
	logLevelError = "[ERROR]"
	logLevelFatal = "[FATAL]"
)

// TrailsLogger implements Logger using log.
type TrailsLogger struct {
	l *log.Logger
}

// NewLogger constructs a TrailsLogger.
func NewLogger() Logger { return &TrailsLogger{log.Default()} }

// Debug writes a debug log.
func (l *TrailsLogger) Debug(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorWhite + msg + colorClose)
		return
	}
	l.l.Printf(colorWhite+msg+" %v"+colorClose+"\n", data)
}

// Debug writes an error log.
func (l *TrailsLogger) Error(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorRed + msg + colorClose)
		return
	}
	l.l.Printf(colorRed+msg+" %v"+colorClose+"\n", data)
}

// Debug writes a fatal log.
func (l *TrailsLogger) Fatal(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorPink+msg+" %v"+colorClose, data)
		return
	}
	l.l.Printf(colorPink+msg+" %v"+colorClose+"\n", data)
}

// Info writes an info log.
func (l *TrailsLogger) Info(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorBlue + msg + colorClose)
		return
	}
	l.l.Printf(colorBlue+msg+" %v"+colorClose+"\n", data)
}

// Warn writes a warning log.
func (l *TrailsLogger) Warn(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorYellow + msg + colorClose)
		return
	}
	l.l.Printf(colorYellow+msg+" %v"+colorClose+"\n", data)
}
