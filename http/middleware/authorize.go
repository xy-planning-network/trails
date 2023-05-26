package middleware

import (
	"net/http"
	"strings"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

// An AuthorizeApplicator constructs Adapters that apply custom authorization rules
// for users, as specified by type T.
type AuthorizeApplicator[T any] struct {
	d *resp.Responder
}

// NewAuthorizeApplicator constructs an AuthorizeApplicator for type T.
// Apply methods for the constructed AuthorizeApplicator will use the Responder for redirects.
// Apply methods will use trails.CurrentUserKey to pull a user out of the request Context.
func NewAuthorizeApplicator[T any](d *resp.Responder) AuthorizeApplicator[T] {
	return AuthorizeApplicator[T]{d}
}

// Apply wraps a custom function validating the authorization of a user,
// whose type is specified by T.
//
// Apply retrieves the value for the trails.CurrentUserKey from the request Context.
// Apply should not be used in a situation where the http.Request.Context
// in some cases stores the requisite value and others does not.
//
// The provided custom function returns either true and an empty string -
// meaning the user is authorized - or false and a valid URL as a string.
//
// If the custom function returns true,
// Apply passes the request to the next handler in the middleware stack.
//
// If the custom function returns false,
// Apply does not pass the request to the next handler in the middleware stack.
//
// Instead, Apply takes one of two actions
// depending on the "Accept" HTTP header of the request.
//   - By default, Apply writes 401.
//   - If "text/html" appears in the "Accept" header, though,
//     Apply sets a "no access" flash on the session
//     and redirects to the URL the custom function returns.
//
// If fn is nil, Apply returns a NoopAdapter.
func (aa AuthorizeApplicator[T]) Apply(fn func(user T) (string, bool)) Adapter {
	if fn == nil {
		return NoopAdapter
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			doRedirect := acceptsTextHtml(r.Header)

			val, ok := r.Context().Value(trails.CurrentUserKey).(T)
			if !ok {
				if doRedirect {
					f := session.Flash{Type: session.FlashWarning, Msg: session.NoAccessMsg}
					if err := aa.d.Redirect(w, r, resp.Flash(f)); err != nil {
						aa.d.Err(w, r, err)
					}

					return
				}

				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if url, ok := fn(val); !ok {
				if doRedirect {
					f := session.Flash{Type: session.FlashWarning, Msg: session.NoAccessMsg}
					if err := aa.d.Redirect(w, r, resp.Flash(f), resp.Url(url)); err != nil {
						aa.d.Err(w, r, err)
					}

					return
				}

				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

// acceptsTextHtml asserts whether the requests accepts rendered HTML or not.
func acceptsTextHtml(header http.Header) bool {
	v := header.Get("Accept")
	if strings.Index(v, "text/html") == 0 {
		return true
	}

	return false
}
