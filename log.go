package trails

import "golang.org/x/exp/slog"

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
