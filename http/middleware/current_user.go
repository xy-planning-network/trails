package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

// The User defines attributes about a user in the context of middleware.
type User interface {
	HasAccess() bool
	HomePath() string
}

// UserStorer defines how to retrieve a User by an ID in the context of middleware.
type UserStorer func(id uint) (User, error)

// CurrentUser pulls the User out of the session.UserSessionable stored in the *http.Request.Context.
//
// A *resp.Responder is needed to handle cases a CurrentUser cannot be retrieved or does not have access.
//
// CurrentUser checks the MIME types of the *http.Request "Accept" header in order to handle
// those cases.
// CurrentUser checks whether the "Accept" MIME type is "application/json"
// and write a status code if so.
// If it isn't, CurrentUser redirects to the Responder's root URL.
func CurrentUser(d *resp.Responder, storer UserStorer, sessionKey, userKey keyring.Keyable) Adapter {
	if d == nil || storer == nil || sessionKey == nil || userKey == nil {
		return NoopAdapter
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, ok := r.Context().Value(sessionKey).(session.TrailsSessionable)
			if !ok {
				handleErr(w, r, http.StatusUnauthorized, d, nil)
				return
			}

			uid, err := s.UserID()
			if err != nil {
				// NOTE(dlk): there is no User in the session,
				// request may be accessing an unauthenticated endpoint,
				// maybe not, something for access control middlewares to determine
				handler.ServeHTTP(w, r)
				return
			}

			user, err := storer(uid)
			if err != nil {
				if err := s.Delete(w, r); err != nil {
					handleErr(w, r, http.StatusInternalServerError, d, err)
					return
				}

				handleErr(w, r, http.StatusUnauthorized, d, err)
				return
			}

			if !user.HasAccess() {
				s.ClearFlashes(w, r)
				if err := s.DeregisterUser(w, r); err != nil {
					handleErr(w, r, http.StatusInternalServerError, d, err)
					return
				}

				handleErr(w, r, http.StatusUnauthorized, d, err)
				return
			}

			if err := s.ResetExpiry(w, r); err != nil {
				s.Delete(w, r) // NOTE(dlk): ignore delete error
				handleErr(w, r, http.StatusInternalServerError, d, err)
				return
			}

			w.Header().Add("Cache-control", "no-store")
			w.Header().Add("Pragma", "no-cache")

			ctx := context.WithValue(r.Context(), userKey, user)
			handler.ServeHTTP(w, r.Clone(ctx))
		})
	}
}

// RequireUnauthed returns a middleware.Adapter that checks whether a user is authenticated
// and requires they not be authenticated.
// When they are not authenticated, RequireUnauthed hands off to the next part of the middleware chain.
//
// Authenticated means a User is set in the request context with the provided key.
//
// When the User is authenticated, and the request's "Accept" header has "application/json" in it,
// RequireUnauthed writes 400 to the client.
// If the request does not have that value in it's header,
// RequireUnauthed redirect to User's HomePath.
func RequireUnauthed(key keyring.Keyable) Adapter {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cu, ok := r.Context().Value(key).(User); ok {
				vs := r.Header.Values("Accept")
				for _, v := range vs {
					if strings.Compare(v, "application/json") == 0 {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				http.Redirect(w, r, cu.HomePath(), http.StatusTemporaryRedirect)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

/// RequireAuthed returns a middleware.Adapter that checks whether a User is authenticated,
// and requires they be authenticated.
// When the User is authenticated, then RequireAuthed hands off to the next part of the middleware chain.
//
// Authenticated means a User is set in the request context with the provided key.
//
// When the User is not authenticated, and the request's "Accept" header has "application/json" in it,
// RequireUnauthed writes 401 to the client.
// If the request does not have that value in it's header,
// RequireAuthed redirects to the provided login URL.
//
// The URL originally requested is appended to as a "next" query param
// when the request method is GET and the endpoint is not the logoff URL.
func RequireAuthed(key keyring.Keyable, loginUrl, logoffUrl string) Adapter {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := r.Context().Value(key).(User); !ok {
				vs := r.Header.Values("Accept")
				for _, v := range vs {
					if strings.Compare(v, "application/json") == 0 {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				}

				u := loginUrl
				if r.Method == http.MethodGet && r.URL.Path != logoffUrl {
					u += "?next=" + url.QueryEscape(r.URL.String())
				}

				http.Redirect(w, r, u, http.StatusTemporaryRedirect)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

// handleErr helps CurrentUser error paths by writing responses reflecting the
// "Accept" type of the *http.Request.
func handleErr(w http.ResponseWriter, r *http.Request, code int, d *resp.Responder, err error) {
	vs := r.Header.Values("Accept")
	for _, v := range vs {
		if strings.Compare(v, "application/json") == 0 {
			d.Json(w, r, resp.Err(err), resp.Code(code))
			return
		}
	}

	d.Redirect(w, r, resp.Err(err))
	return
}
