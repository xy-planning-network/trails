package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/session"
)

func TestInjectSession(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual := middleware.InjectSession(nil, "")

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := r.Context().Value("").(session.Sessionable)
		require.False(t, ok)
		require.Nil(t, val)
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	actual = middleware.InjectSession(stubStore{}, "")

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := r.Context().Value("").(session.Sessionable)
		require.False(t, ok)
		require.Nil(t, val)
	})).ServeHTTP(w, r)

	// Arrange
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	s := stubStore{}
	key := "key"

	// Act
	actual = middleware.InjectSession(s, key)

	// Assert
	actual(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		val, ok := rx.Context().Value(key).(session.Sessionable)
		require.True(t, ok)
		require.NotNil(t, val)
	})).ServeHTTP(w, r)
}

type stubStore struct{}

func (stubStore) GetSession(*http.Request) (session.Sessionable, error) { return session.Stub{}, nil }
