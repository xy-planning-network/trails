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
	cfg := session.Config{
		Env:         trails.Testing,
		SessionName: "Test",
		AuthKey:     notHex,
		EncryptKey:  "",
	}

	// Act
	svc, err := session.NewStoreService(cfg)

	// Assert
	require.NotNil(t, err)
	require.Zero(t, svc)

	// Arrange
	hex := "ABCD"
	cfg.AuthKey = hex
	cfg.EncryptKey = notHex

	// Act
	svc, err = session.NewStoreService(cfg)

	// Assert
	require.NotNil(t, err)
	require.Zero(t, svc)

	// Arrange
	cfg.EncryptKey = hex
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

	//Act
	svc, err = session.NewStoreService(cfg)

	// Assert
	require.Nil(t, err)
	require.NotZero(t, svc)
	require.NotPanics(t, func() { svc.GetSession(r) })
}
