package logger_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/logger"
)

func TestLogContextMarshalText(t *testing.T) {
	// Arrange
	lc := logger.LogContext{}

	// Act
	b, err := lc.MarshalText()

	// Assert
	require.Nil(t, err)
	require.Equal(t, []byte("{}"), b)

	// Arrange
	lc = logger.LogContext{Data: map[string]any{"test": "data"}}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	require.Equal(t, `{"data":{"test":"data"}}`, string(b))

	// Arrange
	lc = logger.LogContext{Error: errors.New("test")}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	require.Equal(t, `{"error":"test"}`, string(b))

	// Arrange
	lc = logger.LogContext{User: testUser{}}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	require.Equal(t, `{"user":{"email":"test@example.com","id":1}}`, string(b))

	// Arrange
	expected := map[string]any{
		"request": map[string]any{
			"method": http.MethodGet,
			"url":    "https://example.com",
			"header": map[string]any{
				"Host": []any{"example.com"},
			},
		},
	}

	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	r.Header.Set("Host", "example.com")
	lc = logger.LogContext{Request: r}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	m := make(map[string]any)
	require.Nil(t, json.Unmarshal(b, &m))
	require.Equal(t, expected, m)

	// Arrange
	expected = map[string]any{
		"request": map[string]any{
			"method": http.MethodPost,
			"url":    "https://example.com/test?some=param",
			"header": map[string]any{
				"Host":         []any{"example.com"},
				"Content-Type": []any{"application/x-www-form-urlencoded"},
			},
		},
	}

	form := url.Values{}
	form.Set("email", "husserl@example.com")
	form.Set("name", "Edmund Husserl")
	s := strings.NewReader(form.Encode())

	r = httptest.NewRequest(http.MethodPost, "https://example.com/test?some=param", s)
	r.Header.Set("Host", "example.com")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()

	lc = logger.LogContext{Request: r}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	m = make(map[string]any)
	require.Nil(t, json.Unmarshal(b, &m))
	require.Equal(t, expected, m)

	// Arrange
	buf := new(bytes.Buffer)
	require.Nil(t, json.NewEncoder(buf).Encode(map[string]string{
		"email": "husserl@example.com",
		"name":  "Edmund Husserl",
	}))
	expected = map[string]any{
		"request": map[string]any{
			"method": http.MethodPost,
			"url":    "https://example.com/test?some=param",
			"header": map[string]any{
				"Host":         []any{"example.com"},
				"Content-Type": []any{"application/json"},
			},
		},
	}

	r = httptest.NewRequest(http.MethodPost, "https://example.com/test?some=param", buf)
	r.Header.Set("Host", "example.com")
	r.Header.Set("Content-Type", "application/json")

	lc = logger.LogContext{Request: r}

	// Act
	b, err = lc.MarshalText()

	// Assert
	require.Nil(t, err)
	m = make(map[string]any)
	require.Nil(t, json.Unmarshal(b, &m))
	require.Equal(t, expected, m)
	_, err = io.ReadAll(r.Body)
	require.Nil(t, err)
}

type testUser struct{}

func (u testUser) GetID() uint      { return 1 }
func (u testUser) GetEmail() string { return "test@example.com" }
