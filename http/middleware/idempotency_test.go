package middleware_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestIdempotent(t *testing.T) {
	// Arrange
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	w := httptest.NewRecorder()

	cache := middleware.NewIdemResMap()

	// Act
	middleware.Idempotent(nil, nil)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusMethodNotAllowed, w.Code)

	// Arrange
	r = httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	w = httptest.NewRecorder()

	// Act
	middleware.Idempotent(cache, sha256.New())(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Arrange
	testKey := "test-idempotency"
	h := sha256.New()
	b := h.Sum(nil)
	h.Reset()

	r = httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	r.Header.Set(middleware.IdempotencyHeader, testKey)

	w = httptest.NewRecorder()

	// Act
	middleware.Idempotent(cache, h)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	v, ok := cache.Get(context.Background(), testKey)
	require.True(t, ok)
	require.Equal(t, http.StatusTeapot, v.Status)
	require.Equal(t, b, v.Req)
	require.Equal(t, new(bytes.Buffer), v.Body)
	require.Equal(t, "/", v.URI)

	// Arrange
	r = httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	r.Header.Set(middleware.IdempotencyHeader, testKey)

	w = httptest.NewRecorder()

	// Act
	middleware.Idempotent(cache, h)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	r = httptest.NewRequest(http.MethodPost, "https://example.com/other", nil)
	r.Header.Set(middleware.IdempotencyHeader, testKey)

	w = httptest.NewRecorder()

	// Act
	middleware.Idempotent(cache, h)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)

	// Arrange
	r = httptest.NewRequest(http.MethodPost, "https://example.com/", strings.NewReader("test"))
	r.Header.Set(middleware.IdempotencyHeader, testKey)

	w = httptest.NewRecorder()

	// Act
	middleware.Idempotent(cache, h)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)

	// Arrange
	otherKey := "other"

	r = httptest.NewRequest(http.MethodPost, "https://example.com/", nil)
	r.Header.Set(middleware.IdempotencyHeader, otherKey)

	w = httptest.NewRecorder()

	cache.Set(r.Context(), otherKey, middleware.NewIdemRes("/", nil))

	// Act
	middleware.Idempotent(cache, h)(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusConflict, w.Code)

	// Arrange
	var incrementMe int
	incrementKey := "increment"
	incrementHandler := http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		incrementMe++
		wx.Write([]byte(strconv.Itoa(incrementMe)))
	})

	for i := 0; i < 3; i++ {
		r = httptest.NewRequest(http.MethodPost, "https://example.com/", nil)
		r.Header.Set(middleware.IdempotencyHeader, incrementKey)

		w = httptest.NewRecorder()

		// Act
		middleware.Idempotent(cache, h)(incrementHandler).ServeHTTP(w, r)

		// Assert
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 1, incrementMe)

		s, err := strconv.Atoi(w.Body.String())
		require.Nil(t, err)
		require.Equal(t, incrementMe, s)
	}
}

func TestNewIdemRes(t *testing.T) {
	// Arrange
	uri := "/test?data=true"
	b := []byte("test")

	// Act
	ir := middleware.NewIdemRes(uri, b)

	// Assert
	require.Equal(t, uri, ir.URI)
	require.Equal(t, b, ir.Req)
	require.Zero(t, ir.Status)
	require.Equal(t, new(bytes.Buffer), ir.Body)
}
