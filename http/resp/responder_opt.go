package resp

import (
	"github.com/xy-planning-network/trails/http/template"
)

type ResponderOptFn func(*Responder)

func WithAuthTemplate(fp string) func(*Responder) {
	return func(d *Responder) {
		d.authed = fp
	}
}

/*
func WithDefaultURL(u string) func(*Responder) {
	return func(d *Responder) {
		d.defaultURL, _ = url.ParseRequestURI(u)
	}
}
*/

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
