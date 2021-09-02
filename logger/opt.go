package logger

import "log"

// A LoggerOptFn is a functional option configuring a TrailsLogger when constructing a new one.
type LoggerOptFn func(*TrailsLogger)

// WithEnv sets the environment TrailsLogger is operating in.
func WithEnv(env string) func(*TrailsLogger) {
	return func(l *TrailsLogger) {
		l.env = env
	}
}

// WithLevel sets the log level TrailsLogger uses.
func WithLevel(level LogLevel) func(*TrailsLogger) {
	return func(l *TrailsLogger) {
		l.ll = level
	}
}

// WithLogger sets the log.Logger TrailsLogger uses.
func WithLogger(log *log.Logger) func(*TrailsLogger) {
	return func(l *TrailsLogger) {
		l.l = log
	}
}
