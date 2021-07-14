package middleware

import (
	"context"
	"net/http"

	"github.com/xy-planning-network/trails/http/ctx"
	"github.com/xy-planning-network/trails/http/session"
)

// InjectSession stores the session associated with the *http.Request in *http.Request.Context.
//
// If store or key are their zero-values, NoopAdapter returns and this middleware does nothing.
func InjectSession(store session.SessionStorer, key ctx.CtxKeyable) Adapter {
	if store == nil || key == nil {
		return NoopAdapter
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, _ := store.GetSession(r)
			ctx := context.WithValue(r.Context(), key, s)
			h.ServeHTTP(w, r.Clone(ctx))
			return
		})
	}
}
