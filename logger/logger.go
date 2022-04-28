package logger

import (
	"log"
	"os"
	"path"
	"regexp"
	"runtime"

	"github.com/fatih/color"
)

const knownFrames = 2

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

// The SkipLogger interface defines a Logger that scrolls back
// the number of frames provided in order to ascertain the call site.
type SkipLogger interface {
	AddSkip(i int) SkipLogger
	Skip() int
	Logger
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
		env: getEnvOrString("ENVIRONEMNT", "DEVELOPMENT"),
		l:   logger,
		ll:  LogLevelInfo,
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

// AddSkip replaces the current number of frames to scroll back
// when logging a message.
//
// Use Skip to get the current skip amount
// when needing to add to it with AddSkip.
func (l *TrailsLogger) AddSkip(i int) SkipLogger {
	newl := *l
	newl.skip = i
	return &newl
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

// Skip returns the current amount of frames to scroll back
// when logging a message.
func (l *TrailsLogger) Skip() int { return l.skip }

// log executes printing the log message,
// including any context if available.
func (l *TrailsLogger) log(colorizer func(string, ...any) string, level LogLevel, msg string, ctx *LogContext) {
	// NOTE(dlk): skip the number of frames the TrailsLogger has
	// and however many the TrailsLogger is configured with
	_, file, line, _ := runtime.Caller(knownFrames + l.skip)

	var toPrint string
	if match := trailsPathRegex.Find([]byte(file)); match != nil {
		toPrint = string(match)
	} else {
		// NOTE(dlk): print the file and the directory it is in
		// e.g.,:
		// /home/dlk/my-project/main.go => my-project/main.go
		// /home/dlk/my-project/internal/internal.go => internal/internal.go
		fullPath, file := path.Split(file)
		toPrint = path.Base(fullPath) + string(os.PathSeparator) + file
	}

	msg = colorizer("%s %s:%d '%s'", level, toPrint, line, msg)
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
