package ctx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/ctx"
)

type testKey string

const (
	sk testKey = "session"
	ck testKey = "currentUserKey"
)

func (tk testKey) Key() string    { return string(tk) }
func (tk testKey) String() string { return string(tk) }

func TestKeyRing(t *testing.T) {
	// Arrange
	kr := ctx.NewKeyRing(nil, nil)

	// Act + Assert
	require.Nil(t, kr)

	// Arrange
	kr = ctx.NewKeyRing(sk, nil)

	// Act + Assert
	require.Nil(t, kr)

	// Arrange
	kr = ctx.NewKeyRing(sk, ck)

	// Act + Assert
	require.Equal(t, sk, kr.SessionKey())
	require.Equal(t, sk, kr.Key(sk.Key()))
	require.Equal(t, ck, kr.CurrentUserKey())
	require.Equal(t, ck, kr.Key(ck.Key()))
}
