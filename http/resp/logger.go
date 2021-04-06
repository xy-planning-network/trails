package resp

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

type logger struct {
	*log.Logger
}

func defaultLogger() *logger { return &logger{log.Default()} }

func (l *logger) Debug(msg string, data map[string]interface{}) {
	l.Printf(colorWhite+msg+colorClose+"\n", data)
}
func (l *logger) Error(msg string, data map[string]interface{}) {
	l.Printf(colorRed+msg+colorClose+"\n", data)
}
func (l *logger) Fatal(msg string, data map[string]interface{}) {
	l.Printf(colorPink+msg+colorClose+"\n", data)
}
func (l *logger) Info(msg string, data map[string]interface{}) {
	l.Printf(colorBlue+msg+colorClose+"\n", data)
}
func (l *logger) Warn(msg string, data map[string]interface{}) {
	l.Printf(colorYellow+msg+colorClose+"\n", data)
}
