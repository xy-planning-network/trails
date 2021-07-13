package middleware

import (
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go/http"
)

// ReportPanic encloses the env and returns a function that when called,
// wraps the passed in http.HandlerFunc in sentryhttp.HandleFunc
// in order to recover and report panics.
func ReportPanic(env string) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		if strings.EqualFold(env, "development") {
			return func(w http.ResponseWriter, r *http.Request) {
				handler(w, r)
			}
		} else {
			sh := sentryhttp.New(sentryhttp.Options{
				Repanic:         false,
				WaitForDelivery: true,
			})
			return sh.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
				handler(w, r)
			})
		}
	}
}
