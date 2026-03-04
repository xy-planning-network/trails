package logger

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/xy-planning-network/trails"
)

var (
	_ encoding.TextMarshaler = LogContext{}
)

// LogUser is the interface exposing attributes of a user to a LogContext.
type LogUser interface {
	// GetID retrieves the application's identifier for a user.
	GetID() int64

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

	// EMF holds the metric definitions if this log entry should also be treated as a CloudWatch metric.
	EMF *EMFMetadata

	// Error is the error that may or may not have instigated a logging event.
	Error error

	// Request is the *http.Request that may or may not have been open during the logging event.
	Request *http.Request

	// LogUser is the user whose session was active during the logging event.
	User LogUser

	env trails.Environment
}

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch_Embedded_Metric_Format_Specification.html
type EMFMetadata struct {
	Namespace  string
	Dimensions []EMFDimension
	Metrics    []EMFMetric
}

type EMFDimension struct {
	Name  string `json:"Name"`
	Value any    `json:"-"`
}

type EMFMetric struct {
	Name  string `json:"Name"`
	Unit  string `json:"Unit"`
	Value any    `json:"-"`
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
		printData := lc.env.IsDevelopment()

		r := make(map[string]any)
		r["method"] = lc.Request.Method

		q := lc.Request.URL.Query()
		trails.Mask(q, "password")
		lc.Request.URL.RawQuery = q.Encode()
		r["url"] = lc.Request.URL.String()

		header := lc.Request.Header
		referer := header.Get("Referer")
		header.Del("Referer")
		if refURL, err := url.ParseRequestURI(referer); err == nil {
			fmt.Fprintln(os.Stderr, refURL.String())
			q := refURL.Query()
			trails.Mask(q, "password")
			refURL.RawQuery = q.Encode()
			header.Set("Referer", refURL.String())
		}

		r["header"] = header
		if ct := lc.Request.Header.Get("Content-Type"); printData && ct == "application/json" {
			j := make(map[string]any)
			b := new(bytes.Buffer)
			tee := io.TeeReader(lc.Request.Body, b)
			if err := json.NewDecoder(tee).Decode(&j); err == nil {
				// FIXME(dlk): We may want to mask values in here.
				// Or, not log them at all.
				// There's a risk of reading the entire JSON blob as a blocking operation
				// in a non-obvious place (i.e., logging).
				r["json"] = j
				lc.Request.Body.Close()
				lc.Request.Body = io.NopCloser(b)
			}
		}

		if printData && lc.Request.Form != nil {
			trails.Mask(lc.Request.Form, "password")
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

	if lc.EMF != nil {
		var dimensions []string

		// 1. Promote Dimensions and Metrics to the root map
		for _, d := range lc.EMF.Dimensions {
			m[d.Name] = d.Value
			dimensions = append(dimensions, d.Name)
		}

		for _, met := range lc.EMF.Metrics {
			m[met.Name] = met.Value
		}

		// 2. Add the AWS Metadata block
		m["_aws"] = map[string]any{
			"Timestamp": time.Now().UnixMilli(),
			"CloudWatchMetrics": []map[string]any{{
				"Namespace":  lc.EMF.Namespace,
				"Dimensions": [][]string{dimensions},
				"Metrics":    lc.EMF.Metrics,
			}},
		}
	}

	return m
}

func processLogValues(m map[string]any) []slog.Attr {
	var g []slog.Attr
	for k, v := range m {
		// NOTE(jlt): Do not group the _aws block.
		// Keep it as a single Attr so it marshals as a JSON object at the root.
		if k == "_aws" {
			g = append(g, slog.Any(k, v))
			continue
		}

		switch t := v.(type) {
		case http.Header:
		// NOTE(dlk): throw away values we don't care to print in application logs.

		case slog.Attr:
			g = append(g, t)

		case map[string]any:
			// NOTE(dlk): break up nested values into slog.Groups
			var subg []any
			for _, val := range processLogValues(t) {
				subg = append(subg, val)
			}

			g = append(g, slog.Group(k, subg...))

		default:
			g = append(g, slog.Any(k, t))
		}
	}
	return g
}
