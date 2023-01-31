package middleware

import (
	"context"
	"net/http"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/resp"
)

// InjectResponder stores a *resp.Responder in the *http.Request.Context
// thereby making it available to handlers.
func InjectResponder(rp *resp.Responder, key trails.Key) Adapter {
	if rp == nil {
		return NoopAdapter
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), key, rp)))
		})
	}
}
