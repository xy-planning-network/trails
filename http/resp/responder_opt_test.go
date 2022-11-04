package resp

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template/templatetest"
	"github.com/xy-planning-network/trails/logger"
)

func TestResponderWithAuthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithAuthTemplate(expected))
	require.Equal(t, expected, d.templates.authed)
}

func TestResponderWithContactErrMsg(t *testing.T) {
	expected := fmt.Sprintf(session.ContactUsErr, "us@example.com")
	d := NewResponder(WithContactErrMsg(expected))
	require.Equal(t, expected, d.contactErrMsg)
}

func TestResponderWithCtxKeys(t *testing.T) {
	tcs := []struct {
		name     string
		keys     []keyring.Keyable
		expected []keyring.Keyable
	}{
		{"nil", nil, nil},
		{"zero-value", make([]keyring.Keyable, 0), nil},
		{"many-zero-value", make([]keyring.Keyable, 99), nil},
		{"sorted", []keyring.Keyable{ctxKey("a"), ctxKey("c"), ctxKey("e"), ctxKey("d")}, []keyring.Keyable{ctxKey("a"), ctxKey("c"), ctxKey("d"), ctxKey("e")}},
		{"deduped", []keyring.Keyable{ctxKey("a"), ctxKey("a"), ctxKey("a")}, []keyring.Keyable{ctxKey("a")}},
		{"filtered-zero-value", []keyring.Keyable{ctxKey(""), ctxKey("a"), ctxKey(""), ctxKey("b"), ctxKey("")}, []keyring.Keyable{ctxKey("a"), ctxKey("b")}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := &Responder{}
			WithCtxKeys(tc.keys...)(d)
			require.Equal(t, tc.expected, d.ctxKeys)
		})
	}
}

func TestResponderWithErrTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithErrTemplate(expected))
	require.Equal(t, expected, d.templates.err)
}

func TestResponderWithLogger(t *testing.T) {
	// Arrange
	b := new(bytes.Buffer)
	l := log.New(b, "", log.LstdFlags)
	ll := logger.New(logger.WithLogger(l))
	d := NewResponder(WithLogger(ll))

	msg := "unit testing is fun!"

	// Act
	d.logger.Info(msg, nil)

	// Assert
	actual := b.String()
	require.Contains(t, actual, "[INFO]")
	require.Contains(t, actual, "responder_opt_test.go")
	require.Contains(t, actual, msg)
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
	expected := ctxKey("test")
	d := NewResponder(WithSessionKey(expected))
	require.Equal(t, expected, d.sessionKey)
}

func TestResponderWithUnauthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithUnauthTemplate(expected))
	require.Equal(t, expected, d.templates.unauthed)
}

func TestResponderWithUserSessionKey(t *testing.T) {
	expected := ctxKey("user")
	d := NewResponder(WithUserSessionKey(expected))
	require.Equal(t, expected, d.userSessionKey)
}

func TestResponderWithVueTemplate(t *testing.T) {
	expected := "vue"
	d := NewResponder(WithVueTemplate(expected))
	require.Equal(t, expected, d.templates.vue)
}

type ctxKey string

func (k ctxKey) Key() string    { return string(k) }
func (k ctxKey) String() string { return string(k) }
