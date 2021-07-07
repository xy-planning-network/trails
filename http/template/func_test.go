package template

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddFn(t *testing.T) {
	// Arrange
	tcs := []struct {
		name   string
		first  string
		second interface{}
		length int
	}{
		{"zero-first", "", nil, 1},
		{"struct-second", "still nil", struct{}{}, 2},
		{"one-good", "one", func() {}, 3},
		{"two-good", "two", func() {}, 4},
		{"repeat", "one", func() {}, 4},
	}

	p := &Parse{}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			require.NotPanics(t, func() { p.AddFn(tc.first, tc.second) })

			// Assert
			if tc.length == 0 {
				require.Nil(t, p.fns)
			} else {
				require.Len(t, p.fns, tc.length)
			}
		})
	}
}

func TestCurrentUser(t *testing.T) {
	// Arrange
	tcs := []struct {
		name     string
		expected interface{}
	}{
		{"nil", nil},
		{"struct", struct{}{}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			name, fn := CurrentUser(tc.expected)

			// Assert
			require.Equal(t, "currentUser", name)
			require.Equal(t, tc.expected, fn())
		})
	}
}

func TestEnv(t *testing.T) {
	// Arrange
	tcs := []struct {
		name     string
		expected string
	}{
		{"zero-value", ""},
		{"env", "testing"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			name, fn := Env(tc.expected)

			// Assert
			require.Equal(t, "env", name)
			require.Equal(t, tc.expected, fn())
		})
	}
}

func TestNonce(t *testing.T) {
	// Arrange + Act
	name, fn := Nonce()

	// Assert
	require.Equal(t, "nonce", name)
	require.NotEqual(t, fn(), fn())

}

func TestRootURL(t *testing.T) {
	// Arrange
	example, err := url.ParseRequestURI("https://example.com")
	require.Nil(t, err)

	tcs := []struct {
		name     string
		actual   *url.URL
		expected string
	}{
		{"nil", nil, ""},
		{"zero-value", new(url.URL), ""},
		{"example.com", example, "https://example.com"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			name, fn := RootURL(tc.actual)

			// Assert
			require.Equal(t, "rootURL", name)
			require.Equal(t, tc.expected, fn())
		})
	}
}
