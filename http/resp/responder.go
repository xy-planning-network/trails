package resp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
)

const responderFrames = 0

// Responder maintains reusable pieces for responding to HTTP requests.
// It exposes many common methods for writing structured data as an HTTP response.
// These are the forms of response Responder can execute:
//
//	Html
//	Json
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
	logger logger.Logger

	// Initialized template parser
	parser template.Parser

	// Pool of *bytes.Buffer to prerender responses into
	pool *sync.Pool

	// Error message to use for "contact us" style client-side error messages,
	// i.e., those set in a session.Flash
	contactErrMsg string

	// Root URL the responder is listening on, also used when in an error state
	rootUrl *url.URL

	// Keys for pulling specific values out of the *http.Request.Context
	ctxKeys []trails.Key

	templates struct {
		// Root template to render when user is authenticated
		authed string

		// Root template to render when an error occurs
		// and no other response can be formed
		err string

		// Root template to render when user is not authenticated
		unauthed string

		// Vue template to render when rendering a Vue app
		vue string
	}
}

// NewResponder constructs a *Responder using the ResponderOptFns passed in.
//
// TODO(dlk): make setting root url required arg? + cannot redirect in err state w/o
func NewResponder(opts ...ResponderOptFn) *Responder {
	// ranging over opts may or may not overwrite defaults
	//
	// TODO(dlk): include default parser?
	d := &Responder{
		pool: &sync.Pool{New: func() any { return new(bytes.Buffer) }},
	}
	for _, opt := range opts {
		opt(d)
	}

	if d.logger == nil {
		d.logger = logger.New()
	}

	if l, ok := d.logger.(logger.SkipLogger); ok {
		d.logger = l.AddSkip(responderFrames)
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
func (doer Responder) CurrentUser(ctx context.Context) (any, error) {
	val := ctx.Value(trails.CurrentUserKey)
	if val == nil {
		return nil, fmt.Errorf("%w: no user found with userSessionKey", ErrNotFound)
	}
	return val, nil
}

// Err wraps http.Error(), logging the error causing the failure state.
//
// Use in exceptional circumstances when no Redirect or Html can occur.
func (doer *Responder) Err(w http.ResponseWriter, r *http.Request, err error, opts ...Fn) {
	rr, nested := doer.do(w, r, append(opts, Err(err))...)
	defer r.Body.Close()
	if nested != nil {
		err = fmt.Errorf("%w: %s", err, nested)
	}

	var msg string
	if err != nil {
		msg = err.Error()
	}

	if rr.code == 0 {
		rr.code = http.StatusInternalServerError
	}

	http.Error(w, msg, rr.code)
}

// Html composes together HTML templates set in *Responder
// and configured by Authed, Unauthed, Tmpls and other such calls.
func (doer *Responder) Html(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, opts...)
	if err != nil {
		return doer.handleHtmlError(w, r, err)
	}

	// TODO(dlk): call Error() instead of silently closing Body?
	if rr.closeBody {
		defer r.Body.Close()
	}

	if doer.parser == nil {
		return doer.handleHtmlError(w, r, fmt.Errorf("%w: no parser configured", ErrBadConfig))
	}

	if len(rr.tmpls) == 0 {
		return doer.handleHtmlError(w, r, fmt.Errorf("%w: no templates to render", ErrMissingData))
	}

	if rr.tmpls[0] == doer.templates.authed {
		// NOTE(dlk): a user is required for an authenticated context.
		// while Authed() also populates the user,
		// this guards against misuse like Html(Tmpls(authedTmpl, otherTmpl)).
		if err := populateUser(*doer, rr); err != nil {
			return doer.handleHtmlError(w, r, err)
		}

		doer.parser.AddFn(template.CurrentUser(rr.user))
	}

	tmpl, err := doer.parser.Parse(rr.tmpls...)
	if err != nil {
		return doer.handleHtmlError(w, r, fmt.Errorf("cannot parse: %w", err))
	}

	rd := struct {
		Data    any
		Flashes []session.Flash
	}{Data: rr.data}

	s, err := doer.Session(r.Context())
	if err != nil && !errors.Is(err, ErrNotFound) {
		return doer.handleHtmlError(w, r, fmt.Errorf("can't retrieve session: %w", err))
	}

	rd.Flashes = s.Flashes(w, r)

	b := doer.pool.Get().(*bytes.Buffer)
	b.Reset()
	defer doer.pool.Put(b)

	if err := tmpl.ExecuteTemplate(b, path.Base(rr.tmpls[0]), rd); err != nil {
		return doer.handleHtmlError(w, r, err)
	}

	if _, err := b.WriteTo(w); err != nil {
		return doer.handleHtmlError(w, r, err)
	}

	return nil
}

type jsonSchema struct {
	D any `json:"data,omitempty"`
	U any `json:"currentUser,omitempty"`
}

// Json responds with data in JSON format, collating it from User(), Data() and setting appropriate headers.
//
// When standard 2xx codes are supplied, the JSON schema will look like this:
//
//	{
//		"currentUser": {},
//		"data": {}
//	}
//
// Otherwise, "currentUser" is elided.
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

	payload := jsonSchema{D: rr.data}
	if rr.code >= http.StatusOK && rr.code <= http.StatusNoContent {
		payload.U = rr.user
	}

	b := doer.pool.Get().(*bytes.Buffer)
	b.Reset()
	defer doer.pool.Put(b)

	if err := json.NewEncoder(b).Encode(payload); err != nil {
		doer.Err(w, r, err)
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(rr.code)
	if _, err := b.WriteTo(w); err != nil {
		return err
	}

	return nil
}

/*
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
// If Url() is not passed in opts, then ToRoot() sets the redirect destination.
//
// The default response status code is 302.
//
// If Code() set the status code to something other than standard redirect 3xx statuses,
// Redirect overwrites the status code with an appropriate 3xx status code.
func (doer *Responder) Redirect(w http.ResponseWriter, r *http.Request, opts ...Fn) error {
	rr, err := doer.do(w, r, append([]Fn{ToRoot()}, opts...)...)
	if err != nil {
		return err
	}

	if rr.closeBody {
		defer r.Body.Close()
	}

	// NOTE(dlk): because of the default ToRoot(),
	// this check safeguards against bugs in the above.
	if rr.url == nil {
		return fmt.Errorf("%w: cannot redirect, no resp.url", ErrMissingData)
	}

	switch {
	case rr.code >= http.StatusMultipleChoices && rr.code <= http.StatusPermanentRedirect:
		// NOTE(dlk): code is already a 3xx, so do nothing
	case rr.code >= http.StatusBadRequest && rr.code < http.StatusInternalServerError:
		// TODO(dlk): use 303?
		rr.code = http.StatusSeeOther
	case rr.code >= http.StatusInternalServerError:
		rr.code = http.StatusTemporaryRedirect
	default:
		rr.code = http.StatusFound
	}

	http.Redirect(w, r, rr.url.String(), rr.code)
	return nil
}

// Session retrieves the session set in the context as a session.Session.
//
// If WithSessionKey was not called setting up the Responder or the context.Context has no
// value for that key, ErrNotFound returns.
func (doer Responder) Session(ctx context.Context) (session.Session, error) {
	val := ctx.Value(trails.SessionKey)
	if val == nil {
		return session.Session{}, fmt.Errorf("%w: no session found with %q", ErrNotFound, trails.SessionKey)
	}

	s, ok := val.(session.Session)
	if !ok {
		return session.Session{}, fmt.Errorf("%w: is not session.Session, is %T", ErrInvalid, val)
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

// handleHtmlError specially renders the error template set on the Responder
// and reports errors.
func (doer *Responder) handleHtmlError(w http.ResponseWriter, r *http.Request, err error) error {
	w.WriteHeader(http.StatusInternalServerError)

	if doer.templates.err == "" {
		err = fmt.Errorf(
			"%w: no error template provided, encountered while handling: %s",
			ErrBadConfig,
			err,
		)
		return err
	}
	b := doer.pool.Get().(*bytes.Buffer)
	b.Reset()
	defer doer.pool.Put(b)

	tmpl, nested := doer.parser.Parse(doer.templates.err)
	if nested != nil {
		err = fmt.Errorf("%w: %s", nested, err)
		doer.logger.Error(err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	nested = tmpl.Execute(b, map[string]any{"Contact": doer.contactErrMsg, "Error": err})
	if nested != nil {
		err = fmt.Errorf("%w: %s", nested, err)
		doer.logger.Error(err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	if _, nested = b.WriteTo(w); nested != nil {
		err = fmt.Errorf("%w: %s", nested, err)
		doer.logger.Error(err.Error(), nil)
		http.Error(w, fmt.Errorf("%w: %s", nested, err).Error(), http.StatusInternalServerError)
		return err
	}

	return nil
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
