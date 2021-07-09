package logger

import "log"

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

type TrailsLogger struct {
	l *log.Logger
}

func DefaultLogger() Logger { return &TrailsLogger{log.Default()} }

func (l *TrailsLogger) Debug(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorWhite + msg + colorClose)
		return
	}
	l.l.Printf(colorWhite+msg+colorClose+"\n", data)
}
func (l *TrailsLogger) Error(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorRed + msg + colorClose)
		return
	}
	l.l.Printf(colorRed+msg+colorClose+"\n", data)
}
func (l *TrailsLogger) Fatal(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorPink+msg+colorClose, data)
		return
	}
	l.l.Printf(colorPink+msg+colorClose+"\n", data)
}
func (l *TrailsLogger) Info(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorBlue + msg + colorClose)
		return
	}
	l.l.Printf(colorBlue+msg+colorClose+"\n", data)
}
func (l *TrailsLogger) Warn(msg string, data map[string]interface{}) {
	if data == nil {
		l.l.Println(colorYellow + msg + colorClose)
		return
	}
	l.l.Printf(colorYellow+msg+colorClose+"\n", data)
}
