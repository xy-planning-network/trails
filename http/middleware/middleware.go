package middleware

import (
	"net/http"
)

// An Adapter enables chaining middlewares together.
type Adapter func(http.Handler) http.Handler

// NoopAdapter is a pass-through Adapter,
// often returned by Adapters available in this package when they are misconfigured.
func NoopAdapter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		return
	})
}

// Chain glues the set of adapters to the handler.
func Chain(handler http.Handler, adapters ...Adapter) http.Handler {
	//NOTE: Loop in reverse to preserve middleware order
	for i := len(adapters) - 1; i >= 0; i-- {
		handler = adapters[i](handler)
	}

	return handler
}
