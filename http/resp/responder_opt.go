package resp

import (
	_ "embed"
	"net/url"

	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
)

// A ResponderOptFn mutates the provided *Responder in some way.
// A ResponderOptFn is used when constructing a new Responder.
type ResponderOptFn func(*Responder)

// WithAdditionalScriptsTemplate sets the template identified by the filepath to use for rendering
// alongside Authed and Unauthed templates.
//
// Authed and Unauthed requires this option.
func WithAdditionalScriptsTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.additionalScripts = fp
	}
}

// WithAuthTemplate sets the template identified by the filepath to use for rendering
// when a user is authenticated.
//
// Authed requires this option.
func WithAuthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.authed = fp
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

// WithErrTemplate sets the template identified by the filepath to use for rendering
// when an unexpected, unhandled error occurs while
func WithErrTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.err = fp
	}
}

// WithLogger sets the provided implementation of Logger in order to log all statements through it.
//
// If no Logger is provided through this option, a defaultLogger will be configured.
func WithLogger(log logger.Logger) func(*Responder) {
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

// WithUnauthTemplate sets the template identified by the filepath to use for rendering
// when a user is not authenticated.
//
// Unauthed requires this option.
func WithUnauthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.unauthed = fp
	}
}

// WithVueTemplate sets the template identified by the filepath to use for rendering
// a Vue client application.
//
// Vue requires this option.
func WithVueTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.vue = fp
	}
}

// WithVueTemplate sets the template identified by the filepath to use for rendering
// additional scripts within a Vue client application.
//
// Vue requires this option.
func WithVueScriptsTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.templates.vueScripts = fp
	}
}
