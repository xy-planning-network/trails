package resp

import (
	"html/template"
	"net/url"
)

type ResponderOptFn func(*Responder)

func WithAuthTemplate(tmpl *template.Template) func(*Responder) {
	return func(d *Responder) {
		d.authed = tmpl
	}
}

func WithDefaultURL(u string) func(*Responder) {
	return func(d *Responder) {
		d.defaultURL, _ = url.ParseRequestURI(u)
	}
}

func WithLogger(logger Logger) func(*Responder) {
	return func(d *Responder) {
		d.Logger = logger
	}
}

func WithSessionKey(key string) func(*Responder) {
	return func(d *Responder) {
		d.sessionKey = key
	}
}

func WithUnauthTemplate(tmpl *template.Template) func(*Responder) {
	return func(d *Responder) {
		d.unauthed = tmpl
	}
}

func WithUserSessionKey(key string) func(*Responder) {
	return func(d *Responder) {
		d.userSessionKey = key
	}
}

func WithVueTemplate(tmpl *template.Template) func(*Responder) {
	return func(d *Responder) {
		d.vue = tmpl
	}
}
