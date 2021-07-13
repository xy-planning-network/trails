package middleware

import (
	"context"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

// RequestID adds a uuid to the request context.
//
// If key is it's zero-value, then NoopAdapter returns and this middleware does nothing.
func RequestID(key string) Adapter {
	if key == "" {
		return NoopAdapter
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), key, uuid.NewV4().String())
			h.ServeHTTP(w, r.Clone(ctx))
		})
	}
}
