package resp

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
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
	// NOTE(dlk): this tests multiple calls to CtxKeys
	// to ensure determinative output
	tcs := []struct {
		name     string
		keys     []trails.Key
		expected []trails.Key
	}{
		{"Nil", nil, []trails.Key(nil)},
		{"Zero-Value", make([]trails.Key, 0), []trails.Key(nil)},
		{"Many-Zero-Value", make([]trails.Key, 99), []trails.Key{}},
		{"Sorted", []trails.Key{"e", "c", "a", "d"}, []trails.Key{"a", "c", "d", "e"}},
		{"Uniqued", []trails.Key{"a", "a", "a"}, []trails.Key{"a"}},
		{"Filtered-Zero-Value", []trails.Key{"", "b", "", "a", ""}, []trails.Key{"a", "b"}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := &Responder{}
			for _, k := range tc.keys {
				WithCtxKeys(k)(d)
			}
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
