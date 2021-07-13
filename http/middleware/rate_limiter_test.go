// +build all rate_limiter

// Include these tests with either -tags all or -tags rate_limiter

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestVisitorFetch(t *testing.T) {
	t.Run("Serial", func(t *testing.T) {
		// Arrange
		vs := middleware.NewVisitors()

		// Act
		v1 := vs.Fetch("127.0.0.1")
		time.Sleep(1 * time.Millisecond)
		v2 := vs.Fetch("127.0.0.1")

		// Assert
		require.Equal(t, v1.Limiter, v2.Limiter)
		require.True(t, v1.LastSeen.Before(v2.LastSeen))

	})

	t.Run("Concurrent", func(t *testing.T) {
		// Arrange
		var wg sync.WaitGroup
		vs := middleware.NewVisitors()
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Act + Assert
				require.NotPanics(t, func() { vs.Fetch("127.0.0.1") })
			}()
		}

		wg.Wait()
	})
}

func TestRateLimit(t *testing.T) {
	// Arrange
	vs := middleware.NewVisitors()
	limiter := middleware.RateLimit(vs)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	limiter(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)

	// Arrange
	vs = middleware.NewVisitors()
	limiter = middleware.RateLimit(vs)

	statuses := make([]int, 21) // NOTE(dlk): one more than burst
	for i := 0; i < 21; i++ {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

		// Act
		limiter(noopHandler()).ServeHTTP(w, r)
		statuses[i] = w.Code
	}

	// Assert
	require.Equal(t, http.StatusTooManyRequests, statuses[len(statuses)-1])

	// Arrange
	time.Sleep(1500 * time.Millisecond) // NOTE(dlk): allow limiter to cooldown
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	// Act
	limiter(teapotHandler()).ServeHTTP(w, r)

	// Assert
	require.Equal(t, http.StatusTeapot, w.Code)
}
