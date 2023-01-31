package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
)

func TestInjectResponder(t *testing.T) {
	// Arrange + Act
	actual := middleware.InjectResponder(nil, trails.Key(""))

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	rp := resp.NewResponder()

	k := trails.Key("testing-inject-responder")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	middleware.InjectResponder(rp, k)(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		actualResponder, ok := rx.Context().Value(k).(*resp.Responder)

		// Assert
		require.True(t, ok)
		require.Equal(t, rp, actualResponder)
	})).ServeHTTP(w, r)
}
