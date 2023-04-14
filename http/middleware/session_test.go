package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/session"
)

func TestInjectSession(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual := middleware.InjectSession(nil)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		_, ok := r.Context().Value("").(session.Session)
		require.False(t, ok)
	})).ServeHTTP(w, r)

	// Arrange
	store := session.NewStub(false)

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual = middleware.InjectSession(store)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		_, ok := r.Context().Value("").(session.Session)
		require.False(t, ok)
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual = middleware.InjectSession(store)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		_, ok := rx.Context().Value(trails.SessionKey).(session.Session)
		require.True(t, ok)
	})).ServeHTTP(w, r)
}
