package middleware

import (
	"context"
	"net/http"
	"uuid"

	"github.com/xy-planning-network/trails"
)

// RequestID adds a UUID to the request context using trails.RequestIDKey.
//
// TODO(dlk): use "X-Request-ID" or similar header for UUID value.
func RequestID() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), trails.RequestIDKey, uuid.New().String())
			*r = *r.Clone(ctx)
			h.ServeHTTP(w, r)
		})
	}
}
