package resp

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/xy-planning-network/trails/http/session"
)

type Responder struct {
	Logger

	// URL to use when in an error state
	defaultURL *url.URL

	// Base template to render when user is authenticated
	authed *template.Template

	// Base template to render when user is not authenticated
	unauthed *template.Template

	// Vue template to render when rendering a Vue app
	vue *template.Template

	// Key for pulling the entire session out of the *http.Request.Context
	sessionKey string

	// Key for pulling the user set in the *http.Request.Context session
	userSessionKey string
}

// NewResponder constructs a *Responder using the ResponderOptFns passed in.
//
// If no logger is provided, a default logger is included.
func NewResponder(opts ...ResponderOptFn) *Responder {
	d := &Responder{Logger: defaultLogger()}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// CurrentUser retrieves the user set in the context.
func (doer Responder) CurrentUser(ctx context.Context) (interface{}, error) {
	val := ctx.Value(doer.userSessionKey)
	if val == nil {
		return nil, fmt.Errorf("%w: no user found with userSessionKey", ErrNotFound)
	}
	return val, nil
}

// Session retrieves the session set in the context.
func (doer Responder) Session(ctx context.Context) (session.Sessionable, error) {
	val := ctx.Value(doer.sessionKey)
	if val == nil {
		return nil, fmt.Errorf("%w: no session found with sessionKey", ErrNotFound)
	}
	return val.(session.Sessionable), nil
}

// Err is a specialized response wrapping http.Error().
//
// Use in exceptional circumstances when no Redirect or Render can occur.
func (doer *Responder) Err(w http.ResponseWriter, r *http.Request, err error) {
	defer r.Body.Close()
	var msg string
	if err != nil {
		msg = err.Error()
	}
	doer.Logger.Error(msg, nil)
	http.Error(w, msg, http.StatusInternalServerError)
}

// Json is a general response with data into JSON from User() and Data() and appropriate headers.
//
// The JSON schema is:
// {
//	"currentUser": {},
//	"data": {}
// }
//
// Data() calls populate "data"
// User() calls populate "currentUser"
func (doer *Responder) Json(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	// TODO(dlk): call Error() instead of silently closing Body?
	if rr.closeBody {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	if rr.code == 0 {
		if err := Code(http.StatusOK)(*doer, rr); err != nil {
			return err
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rr.code)

	b := struct {
		U interface{}            `json:"currentUser,omitempty"`
		D map[string]interface{} `json:"data,omitempty"`
	}{
		D: rr.data,
		U: rr.user,
	}
	if err := json.NewEncoder(w).Encode(b); err != nil {
		return err
	}

	return nil
}

func (doer *Responder) Raw(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	if err != nil {
		r.Body.Close()
		return err
	}

	if rr.closeBody {
		defer r.Body.Close()
	}

	if rr.code == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(rr.code)
	}

	if rr.data != nil {
		// TODO(dlk): encode rr.data as byte stream?
		w.Write(nil)
	}

	return nil
}

// Redirect is a general response calling http.Redirect given Url was called.
//
// If Code was called with something other than 3xx, it is overwritten with an appropriate 3xx status code.
func (doer *Responder) Redirect(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	// TODO(dlk): call Error() instead of silently closing Body?
	if rr.closeBody {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	if rr.url == nil {
		return fmt.Errorf("%w: cannot redirect, no resp.url", ErrMissingData)
	}

	switch {
	case rr.code >= http.StatusBadRequest && rr.code <= http.StatusInternalServerError:
		rr.code = http.StatusSeeOther
	case rr.code > http.StatusInternalServerError:
		rr.code = http.StatusTemporaryRedirect
	default:
		rr.code = http.StatusFound
	}

	http.Redirect(w, r, rr.url.String(), rr.code)
	return nil
}

// Render is a general response composing together HTML templates set in *Responder
// and configured by Authed, Unauthed, Tmpls and other such calls.
func (doer *Responder) Render(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	// TODO(dlk): call Error() instead of silently closing Body?
	if rr.closeBody {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	if len(rr.tmpls) == 0 {
		err := fmt.Errorf("%w: no templates configured for render", ErrMissingData)
		doer.Redirect(w, r, Url(""), GenericErr(err))
		return nil
	}

	/* TODO
	tmpl := template.Must(
		template.New(path.Base(files[0])).
			Funcs(templateFuncs(rr.user, r, h.Env)).
			ParseFiles(files...),
	)
	*/

	s, err := doer.Session(r.Context())
	if err != nil {
		doer.Redirect(
			w, r,
			Url(doer.defaultURL.String()),
			//Warn(session.NoAccessMessage),
			Code(http.StatusUnauthorized),
		)
		return nil
	}

	//rd := struct {
	_ = struct {
		Data    map[string]interface{}
		Flashes []interface{}
	}{
		Data:    rr.data,
		Flashes: s.FetchFlashes(w, r),
	}

	/*
		if err := tmpl.Funcs(template.FuncMap{}).Execute(w, rd); err != nil {
			doer.Err(w, r, err)
			return err
		}
	*/
	return nil
}

// do applies all options to the passed in http.ResponseWriter and *http.Request.
//
// A final terminal option that writes to the http.ResponseWriter concludes the list.
//
// The *http.Request.Body is closed here and cannot be read from again.
//
// Options ought to be passed in the correct order. An option requiring something set by another one should come after.
// do nonetheless attempts to retry calling functional options until all do not return errors or,
// a set of options unable to not return errors is reached.
// Should all options apply successfully, the final terminal option is called.
func (doer *Responder) do(w http.ResponseWriter, r *http.Request, opts ...Fn) (*Response, error) {
	resp := &Response{
		closeBody: true,
		w:         w,
		r:         r,
		data:      make(map[string]interface{}),
		tmpls:     make([]*template.Template, 0),
	}

	var err error
	redos := make([]Fn, 0)
	for _, opt := range opts {
		select {
		case <-r.Context().Done():
			return nil, fmt.Errorf("%w", ErrDone)
		default:
			if err := opt(*doer, resp); err != nil {
				redos = append(redos, opt)
			}
		}
	}

	var i int
	for i < len(redos) {
		select {
		case <-r.Context().Done():
			return nil, fmt.Errorf("%w", ErrDone)
		default:
			// NOTE(dlk): because doer.redo mutates the length of redos,
			// confirm we are running up against a set of functions
			// that will not return anything other than errors by checking
			// the length of redos has not changed since calling doer.redo.
			i = len(redos)
			redos = doer.redo(resp, redos...)
		}
	}

	// NOTE(dlk): wrapup errors to send back
	if len(redos) != 0 {
		for _, opt := range redos {
			err = fmt.Errorf("%w: %s", opt(*doer, resp), err)
		}
	}

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (doer *Responder) redo(r *Response, opts ...Fn) []Fn {
	bad := make([]Fn, 0)
	for _, opt := range opts {
		if err := opt(*doer, r); err != nil {
			bad = append(bad, opt)
		}
	}

	return bad
}
