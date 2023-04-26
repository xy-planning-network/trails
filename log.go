package trails

import (
	"strings"

	"golang.org/x/exp/slog"
)

const (
	LogKindKey = "kind"
	LogMaskVal = "xxxxxx"
)

var (
	AppLogKind    = slog.StringValue("app")
	HTTPLogKind   = slog.StringValue("http")
	WorkerLogKind = slog.StringValue("worker")

	// MaskedLogValue is a convenience [golang.org/x/exp/slog.Value]
	// to be used in implementations of [golang.org/x/exp/slog.LogValuer]
	// to hide sensitive data from log messages.
	MaskedLogValue = slog.StringValue(LogMaskVal)
)

// NewLogLevel translates val into a [golang.org/x/exp/slog.Level]
func NewLogLevel(val string) slog.Level {
	if strings.EqualFold("DEBUG", val) {
		return slog.LevelDebug
	}

	if strings.EqualFold("INFO", val) {
		return slog.LevelInfo
	}

	if strings.EqualFold("WARN", val) {
		return slog.LevelWarn
	}

	if strings.EqualFold("ERROR", val) {
		return slog.LevelError
	}

	return slog.LevelInfo
}
