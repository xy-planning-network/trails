package logger

import (
	"fmt"

	"github.com/getsentry/sentry-go"
)

// A SentryLogger
type SentryLogger struct {
	l Logger
	//ctx LogContext
}

// NewSentryLogger constructs a SentryLogger based off the provided TrailsLogger.
func NewSentryLogger(tl *TrailsLogger, dsn string) Logger {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:          dsn,
		Environment:  tl.env,
		IgnoreErrors: []string{"write: broken pipe"},
	})
	if err != nil {
		err = fmt.Errorf("unable to init Sentry: %s", err)
		tl.Error(err.Error(), nil)
		return tl
	}

	tl.skip = 3
	return &SentryLogger{l: tl}
}

// Debug writes a debug log.
func (sl *SentryLogger) Debug(msg string, ctx *LogContext) {
	sl.l.Debug(msg, ctx)
	//	sl.l.WithContext(sl.ctx).Debug(msg)
	//sl.ctx = LogContext{}
}

// Error writes an error log and sends it to Sentry.
func (sl *SentryLogger) Error(msg string, ctx *LogContext) {
	if sl.l.LogLevel() > LogLevelError {
		return
	}

	//	sl.l.WithContext(sl.ctx).Error(msg)
	sl.l.Error(msg, ctx)
	sl.send(sentry.LevelError, ctx)
	//sl.ctx = LogContext{}
}

// Fatal writes a fatal log and sends it to Sentry.
func (sl *SentryLogger) Fatal(msg string, ctx *LogContext) {
	if sl.l.LogLevel() > LogLevelFatal {
		return
	}

	//	sl.l.WithContext(sl.ctx).Fatal(msg)
	sl.l.Fatal(msg, ctx)
	sl.send(sentry.LevelFatal, ctx)
	//sl.ctx = LogContext{}
}

// Info writes an info log.
func (sl *SentryLogger) Info(msg string, ctx *LogContext) {
	sl.l.Info(msg, ctx)
	//	sl.l.WithContext(sl.ctx).Info(msg)
	//sl.ctx = LogContext{}
}

// Warn writes a warning log and sends it to Sentry.
func (sl *SentryLogger) Warn(msg string, ctx *LogContext) {
	if sl.l.LogLevel() > LogLevelWarn {
		return
	}

	//	sl.l.WithContext(sl.ctx).Warn(msg)
	sl.l.Warn(msg, ctx)
	sl.send(sentry.LevelWarning, ctx)
	//sl.ctx = LogContext{}
}

// LogLevel returns the LogLevel set for the SentryLogger.
func (sl *SentryLogger) LogLevel() LogLevel { return sl.l.LogLevel() }

/*
// WithContext includes the provided LogContext in the next log.
func (sl *SentryLogger) WithContext(ctx LogContext) Logger {
	logger := new(SentryLogger)
	*logger = *sl
	logger.ctx = ctx
	return logger

}
*/

// send ships the LogContext.Error to Sentry,
// including any additional data from LogContext.
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

		scope.SetLevel(level)
		sentry.CaptureException(ctx.Error)
	})
}
