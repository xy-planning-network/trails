package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestRequestID(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual := middleware.RequestID()

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(trails.RequestIDKey).(string)
		require.True(t, ok)
		require.NotZero(t, val)
	})).ServeHTTP(w, r)
}
