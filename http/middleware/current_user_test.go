package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

func TestCurrentUser(t *testing.T) {
	// Arrange + Act
	actual := middleware.CurrentUser(nil, nil, "")

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CurrentUser(resp.NewResponder(), nil, "")

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CurrentUser(resp.NewResponder(), testUserStore(testUser(true)), "")

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com/test")),
		testUserStore(testUser(true)),
		"key",
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "https://example.com/test", w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		testUserStore(testUser(true)),
		"key",
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	key := "key"
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), key, failedSession{errors.New("")}))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		testUserStore(testUser(true)),
		"key",
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value("key").(testUser)
		require.False(t, ok)
		require.False(t, bool(val))
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), key, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		failedUserStore(testUser(true)),
		"key",
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), key, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		testUserStore(testUser(false)),
		"key",
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), key, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		testUserStore(testUser(true)),
		"key",
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(key).(testUser)
		require.True(t, ok)
		require.True(t, bool(val))
	})).ServeHTTP(w, r)

	require.Equal(t, "no-store", w.Header().Get("Cache-control"))
	require.Equal(t, "no-cache", w.Header().Get("Pragma"))
}
