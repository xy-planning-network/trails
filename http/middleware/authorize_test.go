package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
)

func TestAuthorizeApplicator(t *testing.T) {
	// Arrange
	app := middleware.NewAuthorizeApplicator[testUser](nil, nil)

	// Act
	adpt := app.Apply(nil)

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", adpt))

	// Arrange
	sk := ctxKey("session")
	uk := ctxKey("user")
	d := resp.NewResponder(resp.WithSessionKey(sk), resp.WithUserSessionKey(uk))

	app = middleware.NewAuthorizeApplicator[testUser](d, uk)
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
	r = r.WithContext(context.WithValue(r.Context(), uk, testUser(false)))

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.WithContext(context.WithValue(r.Context(), uk, testUser(false)))

	for _, v := range []string{"text/html",
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
	var ss session.Stub

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.WithContext(context.WithValue(r.Context(), sk, ss))
	r = r.WithContext(context.WithValue(r.Context(), uk, testUser(false)))
	r.Header.Set("Accept", "text/html")

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "/oops", w.Header().Get("Location"))

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r = r.WithContext(context.WithValue(r.Context(), uk, testUser(true)))

	// Act
	adpt(teapotHandler()).ServeHTTP(w, r)

	//	Assert
	require.Equal(t, http.StatusTeapot, w.Code)
}
