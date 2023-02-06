package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

func TestCurrentUser(t *testing.T) {
	// Arrange + Act
	actual := middleware.CurrentUser(nil, nil, nil, nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CurrentUser(resp.NewResponder(), nil, nil, nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CurrentUser(resp.NewResponder(), newTestUserStore(true), nil, nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	sessKey := ctxKey("session")
	userKey := ctxKey("user")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com/test")),
		newTestUserStore(true),
		sessKey,
		userKey,
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, "https://example.com/test", w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
		sessKey,
		userKey,
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), sessKey, failedSession{errors.New("")}))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
		sessKey,
		userKey,
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(userKey).(testUser)
		require.False(t, ok)
		require.False(t, bool(val))
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), sessKey, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newFailedUserStore(true),
		sessKey,
		userKey,
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), sessKey, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(false),
		sessKey,
		userKey,
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), sessKey, session.Stub{}))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
		sessKey,
		userKey,
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(userKey).(testUser)
		require.True(t, ok)
		require.True(t, bool(val))
	})).ServeHTTP(w, r)

	require.Equal(t, "no-store", w.Header().Get("Cache-control"))
	require.Equal(t, "no-cache", w.Header().Get("Pragma"))
}

func TestRequireUnauthed(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	actual := middleware.RequireUnauthed(nil)

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	ck := ctxKey("user")
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	actual = middleware.RequireUnauthed(ck)

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	cu := testUser(true)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), ck, cu))

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, cu.HomePath(), w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")
	r = r.Clone(context.WithValue(r.Context(), ck, cu))

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRequireAuthed(t *testing.T) {
	// Arrange
	login := "/login"
	logoff := "/logoff"
	u := "https://example.com"
	q := url.QueryEscape(u)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, u, nil)

	actual := middleware.RequireAuthed(nil, login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, login+"?next="+q, w.Header().Get("Location"))

	// Arrange
	ck := ctxKey("user")
	o := url.QueryEscape("https://example.com/return_to")
	u += "?return_to=" + o
	q = url.QueryEscape(u)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, u, nil)

	actual = middleware.RequireAuthed(ck, login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, login+"?next="+q, w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")

	actual = middleware.RequireAuthed(ck, login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	cu := testUser(true)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), ck, cu))

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)
}
