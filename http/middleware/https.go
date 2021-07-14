package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// ForceHTTPS redirects HTTP requests to HTTPS if the environment is not "development".
//
// The "X-Forwarded-Proto" is used to check whether HTTP was requested due to a trails application
// running behind a proxy.
//
// TODO(dlk): configurable headers to check.
func ForceHTTPS(env string) Adapter {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.EqualFold(env, "development") && r.Header.Get("X-Forwarded-Proto") != "https" {
				u := new(url.URL)
				*u = *r.URL
				u.Scheme = "https"
				u.Host = r.Host

				http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
				return
			}

			handler.ServeHTTP(w, r)
			return
		})
	}
}
