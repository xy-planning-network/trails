package middleware

import (
	"net/http"
	"strings"

	"github.com/xy-planning-network/trails/logger"
)

// LogRequest logs the request's method, requested URL, and originating IP address
// using the enclosed implementation of logger.Logger.
//
// LogRequest scrubs the values for the following keys:
// - password
//
// if logger.Logger is nil, NoopAdapter returns and this middleware does nothing.
func LogRequest(ls logger.Logger) Adapter {
	if ls == nil {
		return NoopAdapter
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := r.URL.Path
			q := r.URL.Query()
			if val := q.Get("password"); val != "" {
				q.Set("password", "xxxxxxx")
			}

			query := q.Encode()
			if query != "" {
				uri += "?" + q.Encode()
			}

			strs := []string{r.Method, uri}
			val := r.Context().Value(IpAddrCtxKey)
			if val != nil {
				strs = append([]string{val.(string)}, strs...)
			}

			ls.Info(strings.Join(strs, " "), nil)
			h.ServeHTTP(w, r)
		})
	}
}
