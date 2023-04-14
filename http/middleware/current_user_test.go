package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

func TestCurrentUser(t *testing.T) {
	// Arrange + Act
	actual := middleware.CurrentUser(nil, nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CurrentUser(resp.NewResponder(), nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com/test")),
		newTestUserStore(true),
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, "/test", w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	s, err := session.NewStub(false).GetSession(r)
	require.Nil(t, err)

	r = r.Clone(context.WithValue(r.Context(), trails.SessionKey, s))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(trails.CurrentUserKey).(testUser)
		require.False(t, ok)
		require.False(t, bool(val))
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	s, err = session.NewStub(true).GetSession(r)
	require.Nil(t, err)

	r = r.Clone(context.WithValue(r.Context(), trails.SessionKey, s))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newFailedUserStore(true),
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	s, err = session.NewStub(true).GetSession(r)
	require.Nil(t, err)

	r = r.Clone(context.WithValue(r.Context(), trails.SessionKey, s))
	r.Header.Set("Accept", "application/json")

	// Act
	middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(false),
	)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	s, err = session.NewStub(true).GetSession(r)
	require.Nil(t, err)

	r = r.Clone(context.WithValue(r.Context(), trails.SessionKey, s))
	r.Header.Set("Accept", "application/json")

	// Act
	actual = middleware.CurrentUser(
		resp.NewResponder(resp.WithRootUrl("https://example.com")),
		newTestUserStore(true),
	)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(trails.CurrentUserKey).(testUser)
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

	actual := middleware.RequireUnauthed()

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	actual = middleware.RequireUnauthed()

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	cu := testUser(true)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, cu))

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, cu.HomePath(), w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, cu))

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

	actual := middleware.RequireAuthed(login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, login+"?next="+q, w.Header().Get("Location"))

	// Arrange
	o := url.QueryEscape("https://example.com/return_to")
	u += "?return_to=" + o
	q = url.QueryEscape(u)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, u, nil)

	actual = middleware.RequireAuthed(login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	require.Equal(t, login+"?next="+q, w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Accept", "application/json")

	actual = middleware.RequireAuthed(login, logoff)

	// Act
	actual(noopHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	cu := testUser(true)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, cu))

	// Act
	actual(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)
}
