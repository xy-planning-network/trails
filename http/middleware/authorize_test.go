package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

func TestAuthorizeApplicator(t *testing.T) {
	// Arrange
	app := middleware.NewAuthorizeApplicator[testUser](nil)

	// Act
	adpt := app.Apply(nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", adpt))

	// Arrange
	d := resp.NewResponder()

	app = middleware.NewAuthorizeApplicator[testUser](d)
	adpt = app.Apply(func(u testUser) (string, bool) {
		if u {
			return "", true
		}

		return "/oops", false
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, testUser(false)))

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, testUser(false)))

	for _, v := range []string{
		"text/html",
		"application/xhtml+xml",
		"application/xml;q=0.9",
		"image/avif",
		"image/webp",
		"*/*",
	} {
		r.Header.Add("Accept", v)
	}

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	ss, err := session.NewStub(false).GetSession(r)
	require.Nil(t, err)

	v := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*"

	r = r.Clone(context.WithValue(r.Context(), trails.SessionKey, ss))
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, testUser(false)))

	r.Header.Set("Accept", v)

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "/oops", w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.Clone(context.WithValue(r.Context(), trails.CurrentUserKey, testUser(true)))

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusTeapot, w.Code)
}
