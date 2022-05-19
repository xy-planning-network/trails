package middleware

import (
	"net/http"
	"strings"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

// An AuthorizeApplicator constructs Adapters that apply custom authorization rules
// for users, as specified by type T.
type AuthorizeApplicator[T any] struct {
	d *resp.Responder
	k keyring.Keyable
}

// NewAuthorizeApplicator constructs an AuthorizeApplicator for type T.
// Apply methods for the constructed AuthorizeApplicator will use the Responder for redirects.
// Apply methods will use the keyring.Keyable to pull a user out of the request Context.
// Accordingly, the keyring.Keyable provided ought to be the same
// as that returned by keyring.CurrentUserKey().
func NewAuthorizeApplicator[T any](d *resp.Responder, k keyring.Keyable) AuthorizeApplicator[T] {
	return AuthorizeApplicator[T]{d, k}
}

// Apply wraps a custom function validating the authorization of a user,
// whose type is specified by T.
//
// Using the kerying.Keyable the AuthorizeApplicator was constructed with,
// Apply retrieves the value for that key from the request Context.
// If it cannot be cast to type T, Apply panics.
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
// - By default, Apply writes 401.
// - If "text/html" appears in the "Accept" header, though,
//   Apply sets a "no access" flash on the session
//   and redirects to the URL the custom function returns.
//
// If fn is nil, Apply returns a NoopAdapter.
func (aa AuthorizeApplicator[T]) Apply(fn func(user T) (string, bool)) Adapter {
	if fn == nil {
		return NoopAdapter
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if url, ok := fn(r.Context().Value(aa.k).(T)); !ok {
				vs := r.Header.Values("Accept")
				for _, v := range vs {
					if strings.Compare(v, "text/html") == 0 {
						f := session.Flash{Type: session.FlashWarning, Msg: session.NoAccessMsg}
						aa.d.Redirect(w, r, resp.Flash(f), resp.Url(url))

						return
					}
				}

				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}
