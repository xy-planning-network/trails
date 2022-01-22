package keyring_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/keyring"
)

type testKey string

const (
	sk testKey = "session"
	ck testKey = "currentUserKey"
	ok testKey = "otherKey"
)

func (tk testKey) Key() string    { return string(tk) }
func (tk testKey) String() string { return string(tk) }

func TestKeyring(t *testing.T) {
	// Arrange
	kr := keyring.NewKeyring(nil, nil)

	// Act + Assert
	require.Nil(t, kr)

	// Arrange
	kr = keyring.NewKeyring(sk, nil)

	// Act + Assert
	require.Nil(t, kr)

	// Arrange
	kr = keyring.NewKeyring(sk, ck)

	// Act + Assert
	require.Equal(t, sk, kr.SessionKey())
	require.Equal(t, sk, kr.Key(sk.Key()))
	require.Equal(t, ck, kr.CurrentUserKey())
	require.Equal(t, ck, kr.Key(ck.Key()))

	// Arrange
	child := keyring.WithKeyring(kr, ok)

	// Act + Assert
	require.Nil(t, kr.Key(ok.Key()))
	require.Equal(t, sk, child.SessionKey())
	require.Equal(t, ck, child.CurrentUserKey())
	require.Equal(t, ok, child.Key(ok.Key()))
}
