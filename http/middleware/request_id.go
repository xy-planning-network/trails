package middleware

import (
	"context"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

const RequestIDCtxKey = "trails/middleware/request-id"

func RequestID() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), RequestIDCtxKey, uuid.NewV4().String())
			h.ServeHTTP(w, r.Clone(ctx))
		})
	}
}
