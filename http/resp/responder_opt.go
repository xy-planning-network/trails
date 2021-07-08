package resp

import (
	"net/url"

	"github.com/xy-planning-network/trails/http/template"
)

type ResponderOptFn func(*Responder)

func WithAuthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.authed = fp
	}
}

func WithCtxInjector(injector ContextInjector) func(*Responder) {
	return func(d *Responder) {
		d.ContextInjector = injector
	}
}

func WithLogger(logger Logger) func(*Responder) {
	return func(d *Responder) {
		d.Logger = logger
	}
}

func WithParser(p template.Parser) func(*Responder) {
	return func(d *Responder) {
		d.parser = p
	}
}

func WithRootURL(u string) func(*Responder) {
	good, err := url.ParseRequestURI(u)
	if err != nil {
		good, _ = url.ParseRequestURI("https://example.com")
	}

	return func(d *Responder) {
		d.rootURL = good

	}
}

func WithSessionKey(key string) func(*Responder) {
	return func(d *Responder) {
		d.sessionKey = key
	}
}

func WithUnauthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.unauthed = fp
	}
}

func WithUserSessionKey(key string) func(*Responder) {
	return func(d *Responder) {
		d.userSessionKey = key
	}
}

func WithVueTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.vue = fp
	}
}
