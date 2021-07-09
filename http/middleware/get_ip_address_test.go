package middleware_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestGetIPAddress(t *testing.T) {
	tcs := []struct {
		name     string
		hm       http.Header
		expected string
	}{
		{"No-Match", make(http.Header), "0.0.0.0"},
		{
			"Only-Private-IP",
			func() http.Header {
				h := make(http.Header)
				h.Set("X-Forwarded-For", "192.168.0.0")
				return h
			}(),
			"0.0.0.0",
		},
		{
			"Only-Public-IP",
			func() http.Header {
				h := make(http.Header)
				h.Set("X-Forwarded-For", "1.1.1.1")
				return h
			}(),
			"1.1.1.1",
		},
		{
			"Get-Before-Proxy",
			func() http.Header {
				h := make(http.Header)
				h.Set("X-Real-Ip", "10.0.0.1,1.1.1.1")
				return h
			}(),
			"1.1.1.1",
		},
		{
			"Get-First-Public",
			func() http.Header {
				h := make(http.Header)
				h.Set("X-Real-Ip", "10.255.255.255,8.8.8.8,1.1.1.1,172.16.0.0")
				return h
			}(),
			"1.1.1.1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, middleware.GetIPAddress(tc.hm))
		})
	}
}
