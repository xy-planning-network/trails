package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestLogRequest(t *testing.T) {
	// Arrange + Act
	actual := middleware.LogRequest(nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	tcs := []struct {
		name     string
		method   string
		ip       string
		url      *url.URL
		expected string
	}{
		{"Zero-Value", http.MethodGet, "", &url.URL{Path: "/"}, "GET /"},
		{"With-IP", http.MethodPost, "127.0.0.1", &url.URL{Path: "/"}, "127.0.0.1 POST /"},
		{
			"With-Query-Params",
			http.MethodPut,
			"0.0.0.0",
			&url.URL{Path: "/hitting/the/trails", RawQuery: "param=true"},
			"0.0.0.0 PUT /hitting/the/trails?param=true",
		},
		{
			"With-Query-Params-Hid",
			http.MethodGet,
			"192.168.0.0",
			&url.URL{Path: "/", RawQuery: "param=true&password=hunter2"},
			"192.168.0.0 GET /?param=true&password=xxxxxxx",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			l := newLogger()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.url.String(), nil)

			if tc.ip != "" {
				r = r.Clone(context.WithValue(r.Context(), middleware.IpAddrCtxKey, tc.ip))
			}

			// Act
			middleware.LogRequest(l)(noopHandler()).ServeHTTP(w, r)

			// Assert
			require.Equal(t, tc.expected, l.String())
		})
	}
}
