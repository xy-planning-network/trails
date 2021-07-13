package resp

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template/templatetest"
	"github.com/xy-planning-network/trails/logger"
)

func TestResponderWithAuthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithAuthTemplate(expected))
	require.Equal(t, expected, d.authed)
}

func TestResponderWithContactErrMsg(t *testing.T) {
	expected := fmt.Sprintf(session.ContactUsErr, "us@example.com")
	d := NewResponder(WithContactErrMsg(expected))
	require.Equal(t, expected, d.contactErrMsg)
}

func TestResponderWithCtxKeys(t *testing.T) {
	tcs := []struct {
		name     string
		keys     []string
		expected []string
	}{
		{"nil", nil, nil},
		{"zero-value", make([]string, 0), nil},
		{"many-zero-value", make([]string, 99), []string{""}},
		{"sorted", []string{"a", "c", "e", "d"}, []string{"a", "c", "d", "e"}},
		{"deduped", []string{"a", "a", "a"}, []string{"a"}},
		{"filtered-zero-value", []string{"", "a", "", "b", ""}, []string{"a", "b"}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := &Responder{}
			WithCtxKeys(tc.keys...)(d)
			require.Equal(t, tc.expected, d.ctxKeys)
		})
	}
}

func TestResponderWithLogger(t *testing.T) {
	l := logger.DefaultLogger()
	d := NewResponder(WithLogger(l))
	require.Equal(t, l, d.Logger)
}

func TestResponderWithParser(t *testing.T) {
	p := templatetest.NewParser()
	d := NewResponder(WithParser(p))
	require.Equal(t, p, d.parser)
}

func TestResponderWithRootUrl(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		u, _ := url.ParseRequestURI("https://example.com")
		expected := u.String()
		d := NewResponder(WithRootUrl("https://example.com"))
		require.Equal(t, expected, d.rootUrl.String())
	})

	t.Run("Null-Byte", func(t *testing.T) {
		expected := "https://example.com"
		d := NewResponder(WithRootUrl(string('\x00')))
		require.Equal(t, expected, d.rootUrl.String())
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
