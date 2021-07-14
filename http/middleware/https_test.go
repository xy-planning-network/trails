package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestForceHTTPS(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	middleware.ForceHTTPS("development")(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("X-Forwarded-Proto", "https")

	// Act
	middleware.ForceHTTPS("testing")(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("X-Forwarded-Proto", "http")

	// Act
	middleware.ForceHTTPS("testing")(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusPermanentRedirect, w.Code)
	require.Contains(t, w.Header().Get("Location"), "https")
}
