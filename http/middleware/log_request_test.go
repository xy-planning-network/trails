package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestLogRequest(t *testing.T) {
	// Arrange + Act
	actual := middleware.LogRequest(nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	ip := "192.168.0.0"
	testID := "test-id"
	useragent := "trails/test"
	content := "very-secret/encoding; shhh"
	referrer := "example.com/referrer"
	respBody := "test"
	newExpected := func(expected middleware.LogRequestRecord) middleware.LogRequestRecord {
		expected.BodySize = len(respBody)
		expected.Host = "example.com"
		expected.ID = testID
		expected.Protocol = "HTTP/1.1"
		expected.Referrer = referrer
		expected.ReqContentType = content
		expected.Status = 200
		expected.UserAgent = useragent

		return expected
	}

	tcs := []struct {
		name     string
		method   string
		ip       string
		url      *url.URL
		expected middleware.LogRequestRecord
	}{
		{
			"Zero-Value",
			http.MethodGet,
			"",
			&url.URL{Path: "/"},
			newExpected(middleware.LogRequestRecord{
				Method: http.MethodGet,
				Path:   "/",
				URI:    "/",
			}),
		},
		{
			"With-IP",
			http.MethodPost,
			ip,
			&url.URL{Path: "/"},
			newExpected(middleware.LogRequestRecord{
				IPAddr: ip,
				Method: http.MethodPost,
				Path:   "/",
				URI:    "/",
			}),
		},
		{
			"With-Query-Params",
			http.MethodPut,
			ip,
			&url.URL{Path: "/hitting/the/trails", RawQuery: "param=true"},
			newExpected(middleware.LogRequestRecord{
				IPAddr: ip,
				Method: http.MethodPut,
				Path:   "/hitting/the/trails",
				URI:    "/hitting/the/trails?param=true",
			}),
		},
		{
			"With-Query-Params-Hid",
			http.MethodGet,
			ip,
			&url.URL{Scheme: "http", Path: "/", RawQuery: "param=true&password=hunter2"},
			newExpected(middleware.LogRequestRecord{
				IPAddr: ip,
				Method: http.MethodGet,
				Path:   "/",
				URI:    "/?param=true&password=" + trails.LogMaskVal,
				Scheme: "http",
			}),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			b := new(bytes.Buffer)
			h := slog.New(slog.NewJSONHandler(b, nil))
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.url.String(), new(bytes.Reader))
			r = r.Clone(context.WithValue(r.Context(), trails.RequestIDKey, "test-id"))

			r.Header.Set("User-Agent", useragent)
			r.Header.Set("Content-Type", content)
			r.Header.Set("Referrer", referrer)

			if tc.ip != "" {
				r = r.Clone(context.WithValue(r.Context(), trails.IpAddrKey, tc.ip))
			}

			var actual middleware.LogRequestRecord

			// Act
			middleware.LogRequest(h)(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
				fmt.Fprint(wx, "test")
			})).ServeHTTP(w, r)

			// Assert
			require.Nil(t, json.Unmarshal(b.Bytes(), &actual))
			require.Equal(t, tc.expected, actual)
		})
	}
}
