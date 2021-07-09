package middleware_test

import (
	"net/http"
)

func NoopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}
