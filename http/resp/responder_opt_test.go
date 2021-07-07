package resp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/template/templatetest"
)

func TestResponderWithAuthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithAuthTemplate(expected))
	require.Equal(t, expected, d.authed)
}

func TestResponderWithLogger(t *testing.T) {
	l := defaultLogger()
	d := NewResponder(WithLogger(l))
	require.Equal(t, l, d.Logger)
}

func TestResponderWithParser(t *testing.T) {
	p := templatetest.NewParser()
	d := NewResponder(WithParser(p))
	require.Equal(t, p, d.parser)
}

func TestResponderWithRootURL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		u, _ := url.ParseRequestURI("https://example.com")
		expected := u.String()
		d := NewResponder(WithRootURL("https://example.com"))
		require.Equal(t, expected, d.rootURL.String())
	})

	t.Run("Null-Byte", func(t *testing.T) {
		expected := "https://example.com"
		d := NewResponder(WithRootURL(string('\x00')))
		require.Equal(t, expected, d.rootURL.String())
	})
}

func TestResponderWithSessionKey(t *testing.T) {
	expected := "test"
	d := NewResponder(WithSessionKey(expected))
	require.Equal(t, expected, d.sessionKey)
}

func TestResponderWithUnauthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithUnauthTemplate(expected))
	require.Equal(t, expected, d.unauthed)
}

func TestResponderWithUserSessionKey(t *testing.T) {
	expected := "user"
	d := NewResponder(WithUserSessionKey(expected))
	require.Equal(t, expected, d.userSessionKey)
}

func TestResponderWithVueTemplate(t *testing.T) {
	expected := "vue"
	d := NewResponder(WithVueTemplate(expected))
	require.Equal(t, expected, d.vue)
}
