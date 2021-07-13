package middleware_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/middleware"
)

func TestCORS(t *testing.T) {
	// Arrange + Act
	actual := middleware.CORS("")

	// Assert
	require.Equal(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))

	// Arrange + Act
	actual = middleware.CORS("https://example.com")

	// Assert
	require.NotEqual(t, fmt.Sprintf("%p", middleware.NoopAdapter), fmt.Sprintf("%p", actual))
}
