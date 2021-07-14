package middleware

import (
	"context"
	"net/http"

	uuid "github.com/satori/go.uuid"
	"github.com/xy-planning-network/trails/http/ctx"
)

// RequestID adds a uuid to the request context.
//
// If key is nil, then NoopAdapter returns and this middleware does nothing.
func RequestID(key ctx.CtxKeyable) Adapter {
	if key == nil {
		return NoopAdapter
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), key, uuid.NewV4().String())
			h.ServeHTTP(w, r.Clone(ctx))
		})
	}
}
