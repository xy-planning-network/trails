package trails_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

func TestByKeyUnique(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    []trails.Key
		expected []trails.Key
	}{
		{"Nil", nil, trails.ByKey{}},
		{"Zero-Value", []trails.Key{}, []trails.Key{}},
		{"None", make([]trails.Key, 0), []trails.Key{}},
		{"Many-Zero", make([]trails.Key, 99), []trails.Key{}},
		{"Sorted", []trails.Key{"a", "c", "e", "d"}, []trails.Key{"a", "c", "d", "e"}},
		{"Uniqued", []trails.Key{"a", "a", "a"}, []trails.Key{"a"}},
		{"Filtered-Zero-Value", []trails.Key{"", "a", "", "b", ""}, []trails.Key{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := trails.ByKey(tc.input).UniqueSort()
			require.Equal(t, tc.expected, []trails.Key(actual))
		})
	}
}
