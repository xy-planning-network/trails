package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

type User interface {
	HasAccess() bool
}

type UserStorer interface {
	GetByID(id uint) (User, error)
}

// CurrentUser pulls the User out of the session.UserSessionable stored in the *http.Request.Context.
//
// A *resp.Responder is needed to handle cases a CurrentUser cannot be retrieved or does not have access.
//
// CurrentUser checks the MIME types of the *http.Request "Accept" header in order to handle
// those cases.
// CurrentUser checks whether the "Accept" MIME type is "application/json"
// and write a status code if so.
// If it isn't, CurrentUser redirects to the Responder's root URL.
func CurrentUser(d *resp.Responder, store UserStorer, key string) Adapter {
	if d == nil || store == nil || key == "" {
		return NoopAdapter
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, ok := r.Context().Value(key).(session.TrailsSessionable)
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

			user, err := store.GetByID(uid)
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

			ctx := context.WithValue(r.Context(), key, user)
			handler.ServeHTTP(w, r.Clone(ctx))
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
