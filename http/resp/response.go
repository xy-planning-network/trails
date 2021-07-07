package resp

import (
	"fmt"
	"net/http"
	"net/url"
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
// If no user can be retrieved from the session, it is assumed a user is not logged in and throws ErrNoUser.
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
// Used with (Responder{}).Render and (Responder{}).Json.
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
		if err := Data(map[string]interface{}{"error": e, "request": r.r})(d, r); err != nil {
			return err
		}

		if e != nil {
			d.Error(e.Error(), r.data)
		}

		if err := Code(http.StatusInternalServerError)(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Flash sets a flash message with the passed in class and msg.
func Flash(class, msg string) Fn {
	return func(d Responder, r *Response) error {
		s, err := d.Session(r.r.Context())
		if err != nil {
			return err
		}

		s.SetFlash(r.w, r.r, class, msg)
		return nil
	}
}

// GenericErr combines Err() and Flash() to log the passed in error and set a generic error flash.
func GenericErr(e error) Fn {
	return func(d Responder, r *Response) error {
		if err := populateUser(d, r); err != nil {
			return err
		}

		if err := Err(e)(d, r); err != nil {
			return err
		}

		// TODO
		// if err := Flash(session.FlashError, errorMessage)(d, r); err != nil {
		if err := Flash("TODO: error class", "TODO: error msg")(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Param adds they query parameter to the response's URL.
//
// Used with (Responder{}).Redirect.
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

// Props first passes a new default initial "initialProps" map into Data then p.
// If the key already exists in the response's data, it's value is overwritten, p being the final map merged.
//
// Used with (Responder{}).Render, specifically in conjunction with Vue.
// Accordingly, prefer this over Data when using Vue.
//
// TODO(dlk): compare usage of vueProps b/w college-try & second-child
// latter uses context injected values and renders them in the final template all as props under "initialProps"
func Props(p map[string]interface{}) Fn {
	return func(d Responder, r *Response) error {
		if err := populateUser(d, r); err != nil {
			return err
		}

		props := map[string]interface{}{
			"initialProps": map[string]interface{}{
				"currentUser": r.user,
			},
		}

		if err := Data(props)(d, r); err != nil {
			return err
		}

		if err := Data(p)(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Success sets the status OK to http.StatusOK and sets a session.FlashSuccess flash with the passed in msg.
//
// Used with (Responder{}).Render.
func Success(msg string) Fn {
	return func(d Responder, r *Response) error {
		if err := Code(http.StatusOK)(d, r); err != nil {
			return err
		}

		// TODO
		// if err := Flash(session.FlashSuccess, msg)(d, r); err != nil {
		if err := Flash("TODO: success class", "TODO: success msg")(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Tmpls appends to the templates to be rendered.
//
// Used with (Responder{}).Render.
func Tmpls(fps ...string) Fn {
	return func(_ Responder, resp *Response) error {
		resp.tmpls = append(resp.tmpls, fps...)
		return nil
	}
}

// Unauthed prepends all templates with the base unauthenticated template.
// If the first template is the base authenticated template, this overwrites it.
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
// Used with (Responder{}).Render and (Responder{}).Json.
// When used with Json, the user is assigned to the "currentUser" key.
func User(u interface{}) Fn {
	return func(d Responder, r *Response) error {
		r.user = u
		return nil
	}
}

// Url parses raw the URL string and sets it in the *Response if successful.
//
// Used with (Responder{}).Redirect.
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
// Used with (Responder{}).Render.
//
// NOTE: Prefer Props over Data to include necessary bits in the Vue template.
func Vue(entry string) Fn {
	return func(d Responder, r *Response) error {
		if err := populateUser(d, r); err != nil {
			return err
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

// Warn sets a flash warning in the session and status code to http.StatusBadRequest.
//
// Used with (Responder{}).Render and (Responder{}).Redirect.
func Warn(msg string) Fn {
	return func(d Responder, r *Response) error {
		if err := Data(map[string]interface{}{"warn": msg, "request": r})(d, r); err != nil {
			return err
		}

		d.Warn(msg, r.data)

		// TODO
		// if err := Flash(session.FlashWarning, msg)(d, r); err != nil {
		if err := Flash("TODO: warn class", msg)(d, r); err != nil {
			return err
		}

		if err := Code(http.StatusBadRequest)(d, r); err != nil {
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
