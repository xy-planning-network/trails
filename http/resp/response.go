package resp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
)

const responseFnFrames = 3

// A Fn is a functional option that mutates the state of the Response.
type Fn func(Responder, *Response) error

// A Response is the internal object a Responder response method builds while applying all
// functional options.
type Response struct {
	w         http.ResponseWriter
	r         *http.Request
	closeBody bool
	code      int
	data      any
	tmpls     []string
	url       *url.URL
	user      any
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

// CurrentUser stores the user in the *Response.
//
// Used with Responder.Html and Responder.Json.
// When used with Json, the user is assigned to the "currentUser" key.
func CurrentUser(u any) Fn {
	return func(d Responder, r *Response) error {
		r.user = u
		return nil
	}
}

// Data stores the provided value for writing to the client.
//
// Used with Responder.Html and Responder.Json.
func Data(d any) Fn {
	return func(_ Responder, r *Response) error {
		r.data = d
		return nil
	}
}

// Err sets the status code http.StatusInternalServerError and logs the error.
func Err(e error) Fn {
	return func(d Responder, r *Response) error {
		if e != nil {
			populateUser(d, r) // NOTE(dlk): ignore err since a user is not required

			u, _ := r.user.(logger.LogUser)
			l := d.logger
			if sl, ok := d.logger.(logger.SkipLogger); ok {
				l = sl.AddSkip(sl.Skip() + responseFnFrames)
			}
			l.Error(e.Error(), newLogContext(r.r, e, r.data, u))
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

		if s == nil {
			return nil
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
		if err := Flash(session.Flash{Type: session.FlashError, Msg: msg})(d, r); err != nil {
			return err
		}

		return nil
	}
}

// Params adds the query parameters to the response's URL.
// Params appends to rather than overwrite other query parameters.
//
// Used with Responder.Redirect.
func Params(pairs map[string]string) Fn {
	return func(_ Responder, r *Response) error {
		if r.url == nil {
			return fmt.Errorf("%w: Url() has not been called", ErrMissingData)
		}

		q := r.url.Query()
		for k, v := range pairs {
			q.Add(k, v)
		}
		r.url.RawQuery = q.Encode()
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

		if err := Flash(session.Flash{Type: session.FlashSuccess, Msg: msg})(d, r); err != nil {
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
		if d.rootUrl == nil {
			return fmt.Errorf("%w: cannot set url, no defined root url", ErrMissingData)
		}
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

// Vue sets a *Response up for rendering a Vue app.
// Vue appends the base Vue template to existing tmpls.
// It adds the required entrypoint to the data to be rendered.

// Vue structures the provided data alongside default values according to a default schema.
//
// Here's the schema:
// {
//	"entry": entry,
//	"props": {
//		"initialProps": {
//			"baseURL": d.rootUrl,
//			"currentUser": r.user,
//		},
//		...key-value pairs set by Data
//		...key-value pairs set by d.ctxKeys
//	},
//	...key-value pairs set by Data
// }
//
// Calls to Data are merged into the required schema in the following way.
//
// At it's simplest, for example, Data(map[string]any{"myProp": "Hello, World"}),
// will produce:
//
// {
//	"entry": entry,
//	"props": {
//		"myProp": "Hello, World",
//		"initialProps": {
//			"baseURL": d.rootUrl,
//			"currentUser": r.user,
//		}
//	}
// }
//
// If the type passed into Data is not map[string]any, Data(myStruct{}),
// the value is placed under another "props" key, producing:
//
// {
//	"entry": entry,
//	"props": {
//		"props": myStruct{},
//		"initialProps": {
//			"baseURL": d.rootUrl,
//			"currentUser": r.user,
//		},
//	}
// }
//
// Finally, if values need to be present to template rendering under a specific key,
// and properties need to be passed in as well,
// include a map[string]any the "initialProps" key
// and the two maps will be merged.
//
// Here's how that's done:
//
// data := map[string]any{
//	"keyForMyTmpl": true,
//	"props": map[string]any{
//		"myProp": "Hello, World"
//	},
// }
// Html(Data(data), Vue(entry))
//
// will produce:
//
// {
//	"entry": entry,
//	"keyForMyTmpl": true
//	"props: {
//		"myProp": "Hello, World",
//		"initialProps": {
//			"baseURL": d.rootUrl,
//			"currentUser": r.user,
//		},
//	},
// }
//
//
// It is not required to set any keys for pulling additional values
// out of the *http.Request.Context.
// Use WithCtxKeys to do so when applicable.
func Vue(entry string) Fn {
	return func(d Responder, r *Response) error {
		if d.vue == "" || entry == "" {
			return nil
		}
		if err := Tmpls(d.vue)(d, r); err != nil {
			return err
		}
		// NOTE(dlk): ignore error since Vue does not require a User
		populateUser(d, r)

		data := map[string]any{"entry": entry}
		init := map[string]any{"currentUser": r.user}
		if d.rootUrl != nil {
			// TODO(dlk): throw error when not configured?
			init["baseURL"] = d.rootUrl.String()
		}

		props := map[string]any{"initialProps": init}
		for _, k := range d.ctxKeys {
			if val := r.r.Context().Value(k); val != nil {
				props[k.Key()] = val
			}
		}

		switch t := r.data.(type) {
		case map[string]any:
			if _, ok := t["props"]; ok {
				// NOTE(dlk): "props" key is set, r.data needs to be merged into
				// both the props map and data map.
				// Perform those checks here and apply key-value pairs accordingly.
				for k, v := range t {
					if k == "props" {
						if ip, ok := v.(map[string]any); ok {
							for k, v := range ip {
								props[k] = v
							}
						}
					} else {
						data[k] = v
					}
				}
			} else {
				// NOTE(dlk): no "props" key was set, apply all to props map.
				for k, v := range t {
					props[k] = v
				}
			}
		default:
			// NOTE(dlk): unhandled case, applying everything to props map under "props" key.
			props["props"] = r.data
		}

		data["props"] = props

		err := Data(data)(d, r)
		if err != nil {
			return err
		}

		return nil
	}
}

// Warn sets a flash warning in the session and logs the warning.
func Warn(msg string) Fn {
	return func(d Responder, r *Response) error {
		populateUser(d, r) // NOTE(dlk): ignore since a user is not required

		u, _ := r.user.(logger.LogUser)
		l := d.logger
		if sl, ok := d.logger.(logger.SkipLogger); ok {
			l = sl.AddSkip(sl.Skip() + responseFnFrames)
		}
		l.Warn(msg, newLogContext(r.r, errors.New(msg), r.data, u))

		if err := Flash(session.Flash{Type: session.FlashWarning, Msg: msg})(d, r); err != nil {
			return err
		}

		return nil
	}
}
