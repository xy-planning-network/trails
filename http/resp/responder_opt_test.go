package resp

import (
	"bytes"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
	"golang.org/x/exp/slog"
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

func TestResponderWithErrTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithErrTemplate(expected))
	require.Equal(t, expected, d.templates.err)
}

func TestResponderWithLogger(t *testing.T) {
	// Arrange
	b := new(bytes.Buffer)
	l := logger.New(slog.New(slog.HandlerOptions{AddSource: true}.NewTextHandler(b)))
	d := NewResponder(WithLogger(l))

	msg := "unit testing is fun!"

	// Act
	d.logger.Info(msg, nil)

	// Assert
	actual := b.String()
	require.Contains(t, actual, "level=INFO")
	require.Contains(t, actual, "responder_opt_test.go")
	require.Contains(t, actual, msg)
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

func TestResponderWithUnauthTemplate(t *testing.T) {
	expected := "test.tmpl"
	d := NewResponder(WithUnauthTemplate(expected))
	require.Equal(t, expected, d.templates.unauthed)
}

func TestResponderWithVueTemplate(t *testing.T) {
	expected := "vue"
	d := NewResponder(WithVueTemplate(expected))
	require.Equal(t, expected, d.templates.vue)
}
