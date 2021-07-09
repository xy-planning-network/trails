package middleware

import (
	"net/http"
)

// An Adapter allows chaining middlewares together.
type Adapter func(http.Handler) http.Handler

// Chain glues the set of adapters to the handler.
func Chain(handler http.Handler, adapters ...Adapter) http.Handler {
	//NOTE: Loop in reverse to preserve middleware order
	for i := len(adapters) - 1; i >= 0; i-- {
		handler = adapters[i](handler)
	}

	return handler
}
