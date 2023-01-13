package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/session"
)

func TestNewService(t *testing.T) {
	// Arrange
	notHex := "😅"

	// Act
	svc, err := session.NewStoreService(trails.Testing, notHex, "", "", "")

	// Assert
	require.NotNil(t, err)
	require.Zero(t, svc)

	// Arrange
	hex := "ABCD"

	// Act
	svc, err = session.NewStoreService(trails.Testing, hex, notHex, "", "")

	// Assert
	require.NotNil(t, err)
	require.Zero(t, svc)

	// Arrange
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	//Act
	svc, err = session.NewStoreService(trails.Testing, hex, hex, "", "")

	// Assert
	require.Nil(t, err)
	require.NotZero(t, svc)
	require.NotPanics(t, func() { svc.GetSession(r) })
}
