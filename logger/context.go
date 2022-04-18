package logger

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	_ encoding.TextMarshaler = LogContext{}
)

type LogUser interface {
	GetID() uint
	GetEmail() string
}

type LogContext struct {
	Data    map[string]any
	Error   error
	Request *http.Request
	User    LogUser
}

// MarshalText converts LogContext into a JSON representation, eliminating zero-value fields.
//
// Values in LogContext.Data that cannot be represented in JSON will cause an error to be thrown.
//
// MarshalText implements encoding.TextMarshaler.
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
