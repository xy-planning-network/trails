package resp

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/xy-planning-network/trails/http/session"
)

// A Fn is a functional option that mutates the state of the Response.
type Fn func(Responder, *Response) error

// A Response is the internal object a Responder response method builds while applying all
// functional options.
//
// Notably, a Response holds a map[string]interface{} that stores data necessary for responding
// to the HTTP request.
type Response struct {
	w         http.ResponseWriter
	r         *http.Request
	closeBody bool
	code      int
	data      map[string]interface{}
	tmpls     []string
	url       *url.URL
	user      interface{}
}

// Authed prepends all templates with the base authenticated template and adds resp.user from the session.
//
// If no user can be retrieved from the session, it is assumed a user is not logged in and returns ErrNoUser.
//
// If WithAuthTemplate was not called setting up the Responder, ErrBadConfig returns.
func Authed() Fn {
	return func(d Responder, r *Response) error {
		if d.authed == "" {
			return fmt.Errorf("%w: no authed tmpl", ErrBadConfig)
		}

		if err := populateUser(d, r); err != nil {
			return err
		}

		if len(r.tmpls) > 0 {
			if r.tmpls[0] == d.authed {
				return nil
			}

			if r.tmpls[0] == d.unauthed {
				r.tmpls[0] = d.authed
				return nil
			}
		}

		r.tmpls = append([]string{d.authed}, r.tmpls...)
		return nil
	}
}

// Code sets the response status code.
func Code(c int) Fn {
	return func(_ Responder, r *Response) error {
		r.code = c
		return nil
	}
}

// Data merges the provided map with existing data.
// If a key already exists, it's value is overwritten.
//
// Used with Responder.Html and Responder.Json.
// When used with Json, this data will populate the "data" key.
//
// Prefer Props over Data when using with Vue if the map provided here is intended to be available
// for rendering the base Vue template.
func Data(d map[string]interface{}) Fn {
	return func(_ Responder, r *Response) error {
		if r.data == nil {
			r.data = make(map[string]interface{})
		}

		for k, v := range d {
			r.data[k] = v
		}

		return nil
	}
}

// Err sets the status code http.StatusInternalServerError and logs the error.
func Err(e error) Fn {
	return func(d Responder, r *Response) error {
		if e != nil {
			logData := map[string]interface{}{"error": e, "request": r.r, "data": r.data}
			d.logger.Error(e.Error(), logData)
		}

		if err := Code(http.StatusInternalServerError)(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Flash sets a flash message in the session with the passed in class and msg.
func Flash(flash session.Flash) Fn {
	return func(d Responder, r *Response) error {
		s, err := d.Session(r.r.Context())
		if err != nil {
			return err
		}

		s.SetFlash(r.w, r.r, flash)
		return nil
	}
}

// GenericErr combines Err() and Flash() to log the passed in error
// and set a generic error flash in the session
// using either the string set by WithContactErrMsg or session.DefaultErrMsg.
func GenericErr(e error) Fn {
	return func(d Responder, r *Response) error {
		if err := Err(e)(d, r); err != nil {
			return err
		}

		msg := session.DefaultErrMsg
		if d.contactErrMsg != "" {
			msg = d.contactErrMsg
		}
		if err := Flash(session.Flash{Class: session.FlashError, Msg: msg})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Param adds they query parameter to the response's URL.
//
// Used with Responder.Redirect.
func Param(key, val string) Fn {
	return func(_ Responder, r *Response) error {
		if r.url == nil {
			return fmt.Errorf("%w: Url() has not been called", ErrMissingData)
		}

		q := r.url.Query()
		q.Add(key, val)
		r.url.RawQuery = q.Encode()
		return nil
	}
}

// Props structures the provided data alongside default values according to a default schema.
//
// Used with Responder.Html, specifically in conjunction with Vue.
// Accordingly, prefer this over Data when using Vue.
//
// Here's the schema:
// {
//	"initialProps": {
//		"baseURL": d.rootUrl,
//		"currentUser": r.user
//		...key-value pairs set by d.ctxKeys
//	},
//	...map set by p
// }
//
// Props first passes p into Data.
// Then, Props passes a new "initialProps" map into Data then p.
//
// It is not required to set any keys for pulling additional values
// out of the *http.Request.Context.
// Use WithCtxKeys to do so when applicable.
func Props(p map[string]interface{}) Fn {
	return func(d Responder, r *Response) error {
		if err := populateUser(d, r); err != nil {
			return err
		}

		if err := Data(p)(d, r); err != nil {
			return err
		}

		// NOTE(dlk): for a configurable approach to this pattern,
		// review https://github.com/xy-planning-network/trails/pull/4
		ip := map[string]interface{}{
			"currentUser": r.user,
			"baseURL":     d.rootUrl,
		}
		for _, k := range d.ctxKeys {
			if val := r.r.Context().Value(k); val != nil {
				ip[k.Key()] = val
			}
		}

		if err := Data(map[string]interface{}{"initialProps": ip})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Success sets the status OK to http.StatusOK
// and sets a session.FlashSuccess flash in the session with the passed in msg.
//
// Used with Responder.Html.
func Success(msg string) Fn {
	return func(d Responder, r *Response) error {
		if err := Code(http.StatusOK)(d, r); err != nil {
			return err
		}

		if err := Flash(session.Flash{Class: session.FlashSuccess, Msg: msg})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Tmpls appends to the templates to be rendered.
//
// Used with Responder.Html.
func Tmpls(fps ...string) Fn {
	return func(_ Responder, r *Response) error {
		r.tmpls = append(r.tmpls, fps...)
		return nil
	}
}

// ToRoot calls URL with the Responder's default, root URL.
func ToRoot() Fn {
	return func(d Responder, r *Response) error {
		r.url = d.rootUrl
		return nil
	}
}

// Unauthed prepends all templates with the base unauthenticated template.
// If the first template is the base authenticated template, this overwrites it.
//
// If WithUnauthTemplate was not called setting up the Responder, ErrBadConfig returns.
func Unauthed() Fn {
	return func(d Responder, r *Response) error {
		if d.unauthed == "" {
			return fmt.Errorf("%w: no unauthed tmpl", ErrBadConfig)
		}

		if len(r.tmpls) > 0 {
			if r.tmpls[0] == d.unauthed {
				return nil
			}

			if r.tmpls[0] == d.authed {
				r.tmpls[0] = d.unauthed
				return nil
			}
		}

		r.tmpls = append([]string{d.unauthed}, r.tmpls...)
		return nil
	}
}

// User stores the user in the *Response.
//
// Used with Responder.Html and Responder.Json.
// When used with Json, the user is assigned to the "currentUser" key.
func User(u interface{}) Fn {
	return func(d Responder, r *Response) error {
		r.user = u
		return nil
	}
}

// Url parses raw the URL string and sets it in the *Response if successful.
//
// Used with Responder.Redirect.
func Url(u string) Fn {
	return func(_ Responder, r *Response) error {
		parsed, err := url.ParseRequestURI(u)
		if err != nil {
			return fmt.Errorf("%w: u is not a valid URL: %v", ErrInvalid, err)
		}
		r.url = parsed
		return nil
	}
}

// Vue appends the base Vue template to existing tmpls.
// It adds the required entrypoint to the data to be rendered.
//
// Used with Responder.Html.
//
// NOTE: Prefer Props over Data to include necessary bits in the Vue template.
func Vue(entry string) Fn {
	return func(d Responder, r *Response) error {
		if d.vue == "" || entry == "" {
			return nil
		}
		if err := Tmpls(d.vue)(d, r); err != nil {
			return err
		}

		if err := Data(map[string]interface{}{"entry": entry})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Warn sets a flash warning in the session and logs the warning.
func Warn(msg string) Fn {
	return func(d Responder, r *Response) error {
		logData := map[string]interface{}{"warn": msg, "request": r.r, "data": r.data}

		d.logger.Warn(msg, logData)

		if err := Flash(session.Flash{Class: session.FlashWarning, Msg: msg})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// populateUser helps pull a user up out of the *Response.r.Context
// and into the *Response itself.
func populateUser(d Responder, r *Response) error {
	if r.user != nil {
		return nil
	}

	u, err := d.CurrentUser(r.r.Context())
	if err != nil || u == nil {
		return ErrNoUser
	}

	return User(u)(d, r)
}
