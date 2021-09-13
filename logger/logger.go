package logger

import (
	"log"
	"os"
	"path"
	"regexp"
	"runtime"

	"github.com/fatih/color"
)

var trailsPathRegex = regexp.MustCompile("trails.*$")

// The Logger interface defines the levels a logging can occur at.
type Logger interface {
	Debug(msg string, ctx *LogContext)
	Error(msg string, ctx *LogContext)
	Fatal(msg string, ctx *LogContext)
	Info(msg string, ctx *LogContext)
	Warn(msg string, ctx *LogContext)

	LogLevel() LogLevel
}

type LogLevel int

const (
	LogLevelUnk LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func NewLogLevel(val string) LogLevel {
	switch val {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelUnk
	}
}

func (ll LogLevel) String() string {
	return map[LogLevel]string{
		LogLevelDebug: "[DEBUG]",
		LogLevelInfo:  "[INFO]",
		LogLevelWarn:  "[WARN]",
		LogLevelError: "[ERROR]",
		LogLevelFatal: "[FATAL]",
		LogLevelUnk:   "[UNK]",
	}[ll]
}

// TrailsLogger implements Logger using log.
type TrailsLogger struct {
	skip int
	env  string
	l    *log.Logger
	ll   LogLevel
}

// NewLogger constructs a TrailsLogger.
//
// Logs are printed to os.Stdout by default, using the std lib log pkg.
// The default environment is DEVELOPMENT.
// The default log level is DEBUG.
func NewLogger(opts ...LoggerOptFn) Logger {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	l := &TrailsLogger{
		skip: 2,
		env:  getEnvOrString("ENVIRONEMNT", "DEVELOPMENT"),
		l:    logger,
		ll:   LogLevelInfo,
	}
	for _, opt := range opts {
		opt(l)
	}

	if sentryDsn := os.Getenv("SENTRY_DSN"); sentryDsn != "" {
		l.Info("SENTRY_DSN set, configuring SentryLogger", nil)
		return NewSentryLogger(l, sentryDsn)
	}

	return l
}

// Debug writes a debug log.
func (l *TrailsLogger) Debug(msg string, ctx *LogContext) {
	if l.ll > LogLevelDebug {
		return
	}

	l.log(color.WhiteString, LogLevelDebug, msg, ctx)
}

// Error writes an error log.
func (l *TrailsLogger) Error(msg string, ctx *LogContext) {
	if l.ll > LogLevelError {
		return
	}

	l.log(color.RedString, LogLevelError, msg, ctx)
}

// Fatal writes a fatal log.
func (l *TrailsLogger) Fatal(msg string, ctx *LogContext) {
	if l.ll > LogLevelFatal {
		return
	}

	l.log(color.MagentaString, LogLevelFatal, msg, ctx)
}

// Info writes an info log.
func (l *TrailsLogger) Info(msg string, ctx *LogContext) {
	if l.ll > LogLevelInfo {
		return
	}

	l.log(color.BlueString, LogLevelInfo, msg, ctx)
}

// Warn writes a warning log.
func (l *TrailsLogger) Warn(msg string, ctx *LogContext) {
	if l.ll > LogLevelWarn {
		return
	}

	l.log(color.YellowString, LogLevelWarn, msg, ctx)
}

// LogLevel returns the LogLevel set for the TrailsLogger.
func (l *TrailsLogger) LogLevel() LogLevel { return l.ll }

/*
// WithContext includes the provided LogContext in the next log.
func (l *TrailsLogger) WithContext(ctx LogContext) Logger {
	logger := new(TrailsLogger)
	*logger = *l
	logger.ctx = ctx
	return logger
}
*/

// log executes printing the log message,
// including any context if available.
func (l *TrailsLogger) log(colorizer func(string, ...interface{}) string, level LogLevel, msg string, ctx *LogContext) {
	// TODO(dlk): have skip be configurable or implement some logic
	// like https://github.com/sirupsen/logrus/blob/b50299cfaaa1bca85be76c8984070e846c7abfd2/entry.go#L178-L213
	_, file, line, _ := runtime.Caller(l.skip)
	if match := trailsPathRegex.Find([]byte(file)); match != nil {
		file = string(match)
	} else {
		file = path.Base(file)
	}

	msg = colorizer("%s %s:%d '%s'", level, file, line, msg)
	if ctx == nil {
		l.l.Println(msg)
		return
	}

	l.l.Println(msg, "log_context:", ctx)
}

func getEnvOrString(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
