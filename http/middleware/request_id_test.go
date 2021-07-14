package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestRequestID(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual := middleware.RequestID(nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value("").(string)
		require.False(t, ok)
		require.Zero(t, val)
	})).ServeHTTP(w, r)

	// Arrange
	key := ctxKey("key")

	// Act
	actual = middleware.RequestID(key)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(key).(string)
		require.True(t, ok)
		require.NotZero(t, val)
	})).ServeHTTP(w, r)
}
