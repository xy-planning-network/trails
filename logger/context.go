package logger

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xy-planning-network/trails"
	"golang.org/x/exp/slog"
)

var (
	_ encoding.TextMarshaler = LogContext{}
)

// LogUser is the interface exposing attributes of a user to a LogContext.
type LogUser interface {
	// GetID retrieves the application's identifier for a user.
	GetID() uint

	// GetEmail retrieves the email address of the user.
	// If not available, an ID should be returned.
	GetEmail() string
}

// A LogContext provides additional information and configuration
// for a [*logger.Logger] method that cannot be tersely captured in the message itself.
type LogContext struct {
	// Caller overrides the caller file and line number with the PC.
	//
	// Caller is not logged in the text of a LogContext.
	//
	// Caller helps goroutines identify the callers of the process that spawned it.
	Caller uintptr

	// Data is any information pertinent at the time of the logging event.
	Data map[string]any

	// Error is the error that may or may not have instigated a logging event.
	Error error

	// Request is the *http.Request that may or may not have been open during the logging event.
	Request *http.Request

	// LogUser is the user whose session was active during the logging event.
	User LogUser
}

func (lc LogContext) LogValue() slog.Value { return slog.GroupValue(lc.attrs()...) }

// MarshalText converts LogContext into a JSON representation,
// eliminating zero-value fields or fields not requiring logging.
//
// Values in LogContext.Data that cannot be represented in JSON will cause an error to be thrown.
//
// MarshalText implements [encoding.TextMarshaler].
func (lc LogContext) MarshalText() ([]byte, error) {
	return json.Marshal(lc.toMap())
}

// String stringifies LogContext as a JSON representation of it.
func (lc LogContext) String() string {
	b, err := json.Marshal(lc)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(b)
}

func (lc LogContext) attrs() []slog.Attr { return processLogValues(lc.toMap()) }

func (lc LogContext) toMap() map[string]any {
	m := make(map[string]any)
	if lc.Data != nil {
		m["data"] = lc.Data
	}

	if lc.Error != nil {
		m["error"] = lc.Error.Error()
	}

	if lc.Request != nil {
		r := make(map[string]any)
		r["method"] = lc.Request.Method
		r["url"] = lc.Request.URL.String()
		r["header"] = lc.Request.Header
		if ct := lc.Request.Header.Get("Content-Type"); ct == "application/json" {
			j := make(map[string]any)
			b := new(bytes.Buffer)
			tee := io.TeeReader(lc.Request.Body, b)
			if err := json.NewDecoder(tee).Decode(&j); err == nil {
				r["json"] = j
				lc.Request.Body.Close()
				lc.Request.Body = io.NopCloser(b)
			}
		}

		if lc.Request.Form != nil {
			r["form"] = lc.Request.Form
		}

		if id, ok := lc.Request.Context().Value(trails.RequestIDKey).(string); ok {
			r["id"] = id
		}

		if len(r) > 0 {
			m["request"] = r
		}
	}

	if lc.User != nil {
		u := make(map[string]any)
		if id := lc.User.GetID(); id != 0 {
			u["id"] = id
		}
		if email := lc.User.GetEmail(); email != "" {
			u["email"] = email
		}
		if len(u) > 0 {
			m["user"] = u
		}
	}

	return m
}

func processLogValues(m map[string]any) []slog.Attr {
	g := make([]slog.Attr, 0)
	for k, v := range m {
		switch t := v.(type) {
		case http.Header:
		// NOTE(dlk): throw away values we don't care to print in application logs.

		case slog.Attr:
			g = append(g, t)

		case map[string]any:
			// NOTE(dlk): break up nested values into slog.Groups
			subg := processLogValues(t)
			g = append(g, slog.Group(k, subg))

		default:
			g = append(g, slog.Any(k, t))
		}
	}
	return g
}
