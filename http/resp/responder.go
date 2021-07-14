package resp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/xy-planning-network/trails/http/ctx"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
)

// Responder maintains reusable pieces for responding to HTTP requests.
// It exposes many common methods for writing structured data as an HTTP response.
// These are the forms of response Responder can execute:
//	Html
// 	Json
//	Redirect
//
// Most oftentimes, setting up a single instance of a Responder suffices for an application.
// Meaning, one needs only application-wide configuration of how HTTP responses should look.
// Our suggestion does not exclude creating diverse Responders
// for non-overlapping segments of an application.
//
// When handling a specific HTTP request, calling code supplies additional data, structure,
// and so forth through Fn functions. While one can create functions of the same type,
// the Responder and Response structs do not expose much - if anything - to interact with.
type Responder struct {
	logger.Logger

	// Initialized template parser
	parser template.Parser

	// Error message to use for "contact us" style client-side error messages,
	// i.e., those set in a session.Flash
	contactErrMsg string

	// Root template to render when user is authenticated
	authed string

	// Root template to render when user is not authenticated
	unauthed string

	// Root URL the responder is listening on, also used when in an error state
	rootUrl *url.URL

	// Keys for pulling specific values out of the *http.Request.Context
	ctxKeys []ctx.CtxKeyable

	// Key for pulling the entire session out of the *http.Request.Context
	sessionKey ctx.CtxKeyable

	// Key for pulling the user set in the *http.Request.Context session
	userSessionKey ctx.CtxKeyable

	// Vue template to render when rendering a Vue app
	vue string
}

// NewResponder constructs a *Responder using the ResponderOptFns passed in.
//
// TODO(dlk): make setting root url required arg? + cannot redirect in err state w/o
func NewResponder(opts ...ResponderOptFn) *Responder {
	// ranging over opts may or may not overwrite defaults
	//
	// TODO(dlk): include default parser?
	d := &Responder{Logger: logger.DefaultLogger()}

	for _, opt := range opts {
		opt(d)
	}

	if d.parser != nil {
		d.parser.AddFn(template.Nonce())
		if d.rootUrl != nil {
			d.parser.AddFn(template.RootUrl(d.rootUrl))
		}
	}

	return d
}

// CurrentUser retrieves the user set in the context.
//
// If WithUserSessionKey was not called setting up the Responder or the context.Context has no
// value for that key, ErrNotFound returns.
func (doer Responder) CurrentUser(ctx context.Context) (interface{}, error) {
	val := ctx.Value(doer.userSessionKey)
	if val == nil {
		return nil, fmt.Errorf("%w: no user found with userSessionKey", ErrNotFound)
	}
	return val, nil
}

// Err wraps http.Error(), logging the error causing the failure state.
//
// Use in exceptional circumstances when no Redirect or Html can occur.
func (doer *Responder) Err(w http.ResponseWriter, r *http.Request, err error, opts ...Fn) {
	rr, nested := doer.do(w, r, opts...)
	defer r.Body.Close()
	if nested != nil {
		err = fmt.Errorf("%w: %s", err, nested)
	}

	var msg string
	if err != nil {
		msg = err.Error()
	}
	doer.Logger.Error(msg, nil)
	if rr.code == 0 {
		rr.code = http.StatusInternalServerError
	}
	http.Error(w, msg, rr.code)
}

// Html composes together HTML templates set in *Responder
// and configured by Authed, Unauthed, Tmpls and other such calls.
func (doer *Responder) Html(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	// TODO(dlk): call Error() instead of silently closing Body?
	if rr.closeBody {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	if doer.parser == nil {
		return fmt.Errorf("%w: no parser configured", ErrBadConfig)
	}

	if len(rr.tmpls) == 0 {
		return fmt.Errorf("%w: no templates to render", ErrMissingData)
	}

	if rr.tmpls[0] == doer.authed {
		// NOTE(dlk): a user is required for an authenticated context
		if err := populateUser(*doer, rr); err != nil {
			return err
		}
		doer.parser.AddFn(template.CurrentUser(rr.user))
	}

	tmpl, err := doer.parser.Parse(rr.tmpls...)
	if err != nil {
		return fmt.Errorf("cannot parse: %w", err)
	}

	// TODO(dlk): necessary to throw error, redirect instead?
	s, err := doer.Session(r.Context())
	if err != nil {
		return fmt.Errorf("can't retrieve session: %w", err)
	}

	rd := struct {
		Data    map[string]interface{}
		Flashes []session.Flash
	}{
		Data:    rr.data,
		Flashes: s.Flashes(w, r),
	}

	if err := tmpl.ExecuteTemplate(w, rr.tmpls[0], rd); err != nil {
		doer.Err(w, r, err)
		return err
	}

	return nil
}

// Json responds with data in JSON format, collating it from User(), Data() and setting appropriate headers.
//
// The JSON schema will look like this:
// {
//	"currentUser": {},
//	"data": {}
// }
//
// User() calls populate "currentUser"
// Data() calls populate "data"
func (doer *Responder) Json(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	// TODO(dlk): call Error() instead of silently closing Body?
	if err != nil {
		return err
	}

	if rr.closeBody {
		defer r.Body.Close()
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

/* TODO(dlk): keep?
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
*/

// Redirect calls http.Redirect, given Url() set the redirect destination.
//
// If Code() set the status code to something other than standard redirect 3xx statuses,
// Redirect overwrites the status code with an appropriate 3xx status code.
func (doer *Responder) Redirect(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, append([]Fn{ToRoot()}, opts...)...)
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

// Session retrieves the session set in the context as a session.FlashSessionable.
//
// session.FlashSessionable identifies the minimal functionality required for resp
// but could be swap out for a more expansive interface such as session.TrailsSessionable.
//
// If WithSessionKey was not called setting up the Responder or the context.Context has no
// value for that key, ErrNotFound returns.
func (doer Responder) Session(ctx context.Context) (session.FlashSessionable, error) {
	val := ctx.Value(doer.sessionKey)
	if val == nil {
		return nil, fmt.Errorf("%w: no session found with sessionKey", ErrNotFound)
	}
	s, ok := val.(session.FlashSessionable)
	if !ok {
		return nil, fmt.Errorf("%w: does not implement session.FlashSessionable", ErrInvalid)
	}
	return s, nil
}

// do applies all options to the passed in http.ResponseWriter and *http.Request.
//
// do closes the *http.Request.Body, which no calling code can read from again.
//
// Calling code ought to pass Options in the correct order.
// An option requiring something set by another one should come after.
// do nonetheless attempts to retry calling functional options until all do not return errors or,
// a set of options unable to not return errors is reached.
//
// Should all options apply successfully, do returns a validly formed *Response.
func (doer *Responder) do(w http.ResponseWriter, r *http.Request, opts ...Fn) (*Response, error) {
	resp := &Response{
		closeBody: true,
		w:         w,
		r:         r,
		data:      make(map[string]interface{}),
		tmpls:     make([]string, 0),
	}

	var err error
	redos := make([]Fn, 0)
	for _, opt := range opts {
		select {
		case <-r.Context().Done():
			return nil, fmt.Errorf("%w", ErrDone)
		default:
			if err = opt(*doer, resp); err != nil {
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
		for i, opt := range redos {
			nested := opt(*doer, resp)
			if i == 0 {
				continue
			}
			err = fmt.Errorf("%w: %s", nested, err)
		}
	}

	if err != nil {
		return resp, err
	}

	return resp, nil
}

// redo applies as many may Options as it can, returning those Options that continue to throw an error.
func (doer *Responder) redo(r *Response, opts ...Fn) []Fn {
	bad := make([]Fn, 0)
	for _, opt := range opts {
		if err := opt(*doer, r); err != nil {
			bad = append(bad, opt)
		}
	}

	return bad
}
