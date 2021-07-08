package resp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultInjector(t *testing.T) {
	// Arrange
	tcs := []struct {
		name     string
		keys     []string
		props    map[string]interface{}
		ctx      context.Context
		expected map[string]interface{}
	}{
		{"both-nil", nil, nil, nil, nil},
		{"ctx-nil", nil, make(map[string]interface{}), nil, make(map[string]interface{})},
		{"keys-nil", nil, make(map[string]interface{}), context.Background(), make(map[string]interface{})},
		{"no-values", []string{"key"}, make(map[string]interface{}), createCtx(nil), make(map[string]interface{})},
		{
			"props-has-values",
			[]string{"key"},
			map[string]interface{}{"test": 1},
			createCtx(nil),
			map[string]interface{}{"test": 1},
		},
		{
			"ctx-adds-values",
			[]string{"key"},
			map[string]interface{}{"test": 1},
			createCtx([]string{"key"}),
			map[string]interface{}{"key": 0, "test": 1},
		},
		{
			"ctx-overwrites",
			[]string{"test"},
			map[string]interface{}{"test": 1},
			createCtx([]string{"test"}),
			map[string]interface{}{"test": 0},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			i := DefaultInjector{tc.keys}

			// Act
			require.NotPanics(t, func() { i.Inject(tc.props, tc.ctx) })

			// Assert
			require.Equal(t, tc.expected, tc.props)
		})
	}
}

func createCtx(keys []string) context.Context {
	ctx := context.Background()
	for i, k := range keys {
		ctx = context.WithValue(ctx, k, i)
	}
	return ctx
}
