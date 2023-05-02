package logger

import (
	"context"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/exp/slog"
)

const (
	callerTmpl  = "%s:%d"
	knownFrames = 2

	ansiRed    = "\033[91m"
	ansiYellow = "\033[93m"
	ansiBlue   = "\033[94m"
	ansiReset  = "\033[0m"
)

var (
	emptyJSON       = []byte(`"{}"`)
	trailsPathRegex = regexp.MustCompile("/trails/.*$")
)

// The Logger interface defines the ways of logging messages at certain levels of importance.
type Logger interface {
	// AddSkip sets the number of stacktrace frames to ascend when
	// determining the file and line number of the log message.
	//
	// NB: AddSkip does not add to the amount set by previous AddSkip calls.
	// To add to it, do something like: l = l.AddSkip(l.Skip + i)
	AddSkip(i int) Logger

	// Skip returns the number of frames that not be included
	// when determining the file/line number of a call to a log message method.
	Skip() int

	// Debug writes a debug log message.
	Debug(msg string, ctx *LogContext)

	// Error writes an error log message.
	Error(msg string, ctx *LogContext)

	// Info writes an info log message.
	Info(msg string, ctx *LogContext)

	// Warn writes a warning log message.
	Warn(msg string, ctx *LogContext)
}

// TrailsLogger implements [Logger] using [golang.org/x/exp/slog.Logger].
type TrailsLogger struct {
	l    *slog.Logger
	skip int
}

// New constructs a Logger using [golang.org/x/exp/slog.Logger].
func New(log *slog.Logger) Logger { return &TrailsLogger{l: log} }

func (l *TrailsLogger) AddSkip(i int) Logger {
	newl := *l
	newl.skip = i
	return &newl
}

func (l *TrailsLogger) Skip() int                         { return l.skip }
func (l *TrailsLogger) Debug(msg string, ctx *LogContext) { l.log(slog.LevelDebug, msg, ctx) }
func (l *TrailsLogger) Error(msg string, ctx *LogContext) { l.log(slog.LevelError, msg, ctx) }
func (l *TrailsLogger) Info(msg string, ctx *LogContext)  { l.log(slog.LevelInfo, msg, ctx) }
func (l *TrailsLogger) Warn(msg string, ctx *LogContext)  { l.log(slog.LevelWarn, msg, ctx) }

// log executes printing the log message,
// including any context if available.
func (l *TrailsLogger) log(level slog.Level, msg string, ctx *LogContext) {
	if ctx == nil {
		ctx = new(LogContext)
	}

	pc := ctx.Caller
	if pc == 0 {
		// NOTE(dlk): skip the number of frames the TrailsLogger has
		// and however many the TrailsLogger is configured with
		pc, _, _, _ = runtime.Caller(knownFrames + l.Skip())
	}

	rec := slog.NewRecord(time.Now(), level, msg, pc)
	rec.AddAttrs(ctx.attrs()...)

	l.l.Handler().Handle(context.TODO(), rec)
}

// ColorizeLevel adds color to the log level!
func ColorizeLevel(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		switch slog.Level(a.Value.Int64()) {
		case slog.LevelDebug:
			a.Value = slog.StringValue("[DEBUG]")
		case slog.LevelInfo:
			a.Value = slog.StringValue(ansiBlue + "[INFO]" + ansiReset)
		case slog.LevelWarn:
			a.Value = slog.StringValue(ansiYellow + "[WARN]" + ansiReset)
		case slog.LevelError:
			a.Value = slog.StringValue(ansiRed + "[ERROR]" + ansiReset)
		}
	}

	return a
}

// DeleteLevelAttr removes the log level from output.
func DeleteLevelAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		a = slog.Attr{}
	}

	return a
}

// DeleteMessageAttr removes the message from output.
func DeleteMessageAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.MessageKey {
		a = slog.Attr{}
	}

	return a
}

// TruncSourceAttr truncates the full filepath of the source log call
// to a more-to-the-point path.
func TruncSourceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		var val string
		switch v := a.Value.Any().(type) {
		case runtime.Frame: //NOTE(dlk): github.com/xy-planning-network/tint
			val = immediateFilepath(v.File)
			val += ":" + strconv.Itoa(v.Line)

		case string: //NOTE(dlk): golang.org/x/exp/slog
			val = immediateFilepath(v)
		}

		a = slog.Attr{Key: slog.SourceKey, Value: slog.StringValue(val)}
	}

	return a
}

// immediateFilepath either shortens a full filepath to the most immediate parent directory and file,
// or returns the full trails package path; e.g.,
//
// /path/to/my-project/main.go => my-project/main.go
//
// /path/to/trails/http/resp/responder.go => trails/http/resp/responder.go
func immediateFilepath(file string) string {
	if match := trailsPathRegex.Find([]byte(file)); match != nil {
		return string(match[1:])
	}

	fullPath, file := path.Split(file)
	return path.Base(fullPath) + string(os.PathSeparator) + file
}
