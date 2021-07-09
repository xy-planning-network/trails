package middleware_test

import (
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
			go func() {
				wg.Add(1)
				defer wg.Done()

				// Act
				require.NotPanics(t, func() { vs.Fetch("127.0.0.1") })
			}()
		}

		wg.Wait()
	})
}

func TestRateLimit(t *testing.T) {
	// TODO(dlk): unit test GetIPAddress

}
