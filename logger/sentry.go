package logger

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/xy-planning-network/trails"
	"golang.org/x/exp/slog"
)

const (
	knownSentryLogFrames     = 1
	knownSentryCaptureFrames = 2
)

// A SentryLogger logs messages and reports sufficiently important
// ones to error tracking software Sentry (https://sentry.io).
type SentryLogger struct {
	l Logger
}

// NewSentryLogger constructs a [*SentryLogger] based off the provided [*TrailsLogger],
// routing messages to the DSN provided.
func NewSentryLogger(env trails.Environment, l Logger, dsn string) Logger {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:          dsn,
		Environment:  env.String(),
		IgnoreErrors: []string{"write: broken pipe"},
	})
	if err != nil {
		err = fmt.Errorf("unable to init Sentry: %s", err)
		l.Error(err.Error(), nil)

		return nil
	}
	l.Debug("initing SentryLogger", nil)

	return &SentryLogger{l: l.AddSkip(l.Skip() + knownSentryLogFrames)}
}

// Unwrap exposes the underlying Logger backing the *SentryLogger.
func (sl *SentryLogger) Unwrap() Logger { return sl.l }

func (sl *SentryLogger) AddSkip(i int) Logger { return &SentryLogger{sl.l.AddSkip(i)} }

func (sl *SentryLogger) Skip() int { return sl.l.Skip() }

// Debug writes a debug log.
func (sl *SentryLogger) Debug(msg string, ctx *LogContext) {
	sl.l.Debug(msg, ctx)
}

// Error writes an error log and sends it to Sentry.
func (sl *SentryLogger) Error(msg string, ctx *LogContext) {
	if tl, ok := sl.l.(*TrailsLogger); ok && !tl.l.Enabled(nil, slog.LevelError) {
		return
	}

	sl.l.Error(msg, ctx)
	sl.send(sentry.LevelError, ctx)
}

// Info writes an info log.
func (sl *SentryLogger) Info(msg string, ctx *LogContext) {
	sl.l.Info(msg, ctx)
}

// Warn writes a warning log and sends it to Sentry.
func (sl *SentryLogger) Warn(msg string, ctx *LogContext) {
	if tl, ok := sl.l.(*TrailsLogger); ok && !tl.l.Enabled(nil, slog.LevelWarn) {
		return
	}

	sl.l.Warn(msg, ctx)
	sl.send(sentry.LevelWarning, ctx)
}

// send ships the *LogContext.Error to Sentry,
// including any additional data from *LogContext.
func (sl *SentryLogger) send(level sentry.Level, ctx *LogContext) {
	if ctx == nil || ctx.Error == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		if ctx.User != nil {
			scope.SetUser(sentry.User{
				Email: ctx.User.GetEmail(),
				ID:    fmt.Sprint(ctx.User.GetID()),
			})
		}

		if ctx.Request != nil {
			scope.SetRequest(ctx.Request)
		}

		if ctx.Data != nil {
			scope.SetExtra("data", ctx.Data)
		}

		scope.AddEventProcessor(skipBackFrames(sl.Skip()))
		scope.SetLevel(level)

		sentry.CaptureException(ctx.Error)
	})
}

// skipBackFrames removes stacktrace frames from the *sentry.Event
// so Sentry accurately captures the immediately relevant line of code
// as the source of the exception.
func skipBackFrames(skip int) func(*sentry.Event, *sentry.EventHint) *sentry.Event {
	return func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		for i, exc := range event.Exception {
			// NOTE(dlk): strip out frames from sentry pkg (knownSentryCaptureFrames)
			// and any additional frames the Logger was setup with,
			// as identified by skip.
			last := len(exc.Stacktrace.Frames) - knownSentryCaptureFrames - skip
			event.Exception[i].Stacktrace.Frames = exc.Stacktrace.Frames[:last]
		}
		return event
	}
}
