package middleware

import (
	"context"
	"net/http"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/session"
)

// InjectSession stores the session associated with the *http.Request in *http.Request.Context.
//
// If store is its zero-value, NoopAdapter returns and this middleware does nothing.
func InjectSession(store session.SessionStorer) Adapter {
	if store == nil {
		return NoopAdapter
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, _ := store.GetSession(r)

			ctx := context.WithValue(r.Context(), trails.SessionKey, s)
			*r = *r.Clone(ctx)

			s.Save(w, r)
			h.ServeHTTP(w, r)

			return
		})
	}
}
