package middleware

import (
	"net/http"

	"github.com/gorilla/handlers"
)

// CORS sets "Access-Control-Allowed" style headers on a response.
// The handler including this middleware must also handle the http.MethodOptions method
// and not just the HTTP method it's designed for.
func CORS(base string) Adapter {
	return handlers.CORS(
		handlers.AllowedHeaders([]string{
			"Content-Type",
			"X-CSRF-Token",
		}),
		handlers.AllowedOrigins([]string{base}),
		handlers.AllowedMethods([]string{
			http.MethodDelete,
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
			http.MethodPost,
			http.MethodPut,
		}),
	)
}
