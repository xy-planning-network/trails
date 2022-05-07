package resp

import (
	"net/url"
	"sort"

	"github.com/xy-planning-network/trails/http/ctx"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
)

// A ResponderOptFn mutates the provided *Responder in some way.
// A ResponderOptFn is used when constructing a new Responder.
type ResponderOptFn func(*Responder)

// NoopResponderOptFn is a pass-through ResponderOptFn,
// often returned by other ResponderOptFns when they are called incorrectly.
func NoopResponderOptFn(_ *Responder) {}

// WithAuthTemplate sets the template identified by the filepath to use for rendering
// when a user is authenticated.
//
// Authed requires this option.
func WithAuthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.authed = fp
	}
}

// WithContactErrMsg sets the error message to use for error Flashes.
//
// We recommend using session.ContactUsErr as a template.
func WithContactErrMsg(msg string) func(*Responder) {
	return func(d *Responder) {
		d.contactErrMsg = msg
	}
}

// WithCtxKeys appends the provided keys to be used for retrieving values from the *http.Request.Context.
//
// WithCtxKeys deduplicates keys and filters out zero-value strings.
func WithCtxKeys(keys ...ctx.CtxKeyable) func(*Responder) {
	if len(keys) == 0 {
		return NoopResponderOptFn
	}
	return func(d *Responder) {
		for _, k := range keys {
			if k == nil {
				continue
			}
			d.ctxKeys = append(d.ctxKeys, k)
		}

		// NOTE(dlk): filter and deduplicate strings
		// cribbed from: https://github.com/golang/go/wiki/SliceTricks#in-place-deduplicate-comparable
		sort.Sort(ctx.ByCtxKeyable(d.ctxKeys))
		j := 0
		for i := 1; i < len(d.ctxKeys); i++ {
			switch d.ctxKeys[j].String() {
			case d.ctxKeys[i].String():
				continue
			case "":
				d.ctxKeys[j] = d.ctxKeys[i]
				continue
			default:
				j++
				d.ctxKeys[j] = d.ctxKeys[i]
			}
		}
		if len(d.ctxKeys) == 0 {
			return
		}
		d.ctxKeys = d.ctxKeys[:j+1]
	}
}

// WithLogger sets the provided implementation of Logger in order to log all statements through it.
//
// If no Logger is provided through this option, a defaultLogger will be configured.
func WithLogger(log logger.Logger) func(*Responder) {
	if log == nil {
		log = logger.New()
	}
	return func(d *Responder) {
		d.logger = log
	}
}

// WithParser sets the provided implementation of template.Parser to use for parsing HTML templates.
func WithParser(p template.Parser) func(*Responder) {
	return func(d *Responder) {
		d.parser = p
	}
}

// WithRootUrl sets the provided URL after parsing it into a *url.URL to use for rendering and redirecting
//
// NOTE: If u fails parsing by url.ParseRequestURI, the root URL becomes https://example.com
func WithRootUrl(u string) func(*Responder) {
	good, err := url.ParseRequestURI(u)
	if err != nil {
		good, _ = url.ParseRequestURI("https://example.com")
	}

	return func(d *Responder) {
		d.rootUrl = good

	}
}

// WithSessionKey sets the key to use for grabbing a session.Sessionable out of the *http.Request.Context
//
// Responder.Session requires this option.
func WithSessionKey(key ctx.CtxKeyable) func(*Responder) {
	return func(d *Responder) {
		d.sessionKey = key
	}
}

// WithUnauthTemplate sets the template identified by the filepath to use for rendering
// when a user is not authenticated.
//
// Unauthed requires this option.
func WithUnauthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.unauthed = fp
	}
}

// WithUserSessionKey sets the key to use for grabbing a user
// out of the session.Sessionable set in the *http.Request.Context
//
// Responder.CurrentUser requires this option.
func WithUserSessionKey(key ctx.CtxKeyable) func(*Responder) {
	return func(d *Responder) {
		d.userSessionKey = key
	}
}

// WithVueTemplate sets the template identified by the filepath to use for rendering
// a Vue client application.
//
// Vue requires this option.
func WithVueTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.vue = fp
	}
}
