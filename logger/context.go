package logger

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
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
	// Caller overrides the caller file and line number with the provided value.
	//
	// Caller is not logged in the text of a LogContext.
	//
	// Caller helps goroutines identify the callers of the process that spawned it.
	Caller string

	// Data is any information pertinent at the time of the logging event.
	Data map[string]any

	// Error is the error that may or may not have instigated a logging event.
	Error error

	// Request is the *http.Request that may or may not have been open during the logging event.
	Request *http.Request

	// LogUser is the user whose session was active during the logging event.
	User LogUser
}

// MarshalText converts LogContext into a JSON representation,
// eliminating zero-value fields or fields not requiring logging.
//
// Values in LogContext.Data that cannot be represented in JSON will cause an error to be thrown.
//
// MarshalText implements [encoding.TextMarshaler].
func (lc LogContext) MarshalText() ([]byte, error) {
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

	return json.Marshal(m)
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

// CurrentCaller retrieves the caller for the caller of CurrentCaller,
// formatted for using as a value in LogContext.Caller.
//
//  myFunc() { 		<- returns this caller
//		func() {
//			CurrentCaller()
//		}()
//  }
func CurrentCaller() string {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Sprintf(callerTmpl, immediateFilepath(file), line)
}
