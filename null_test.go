package trails_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

func TestNullTimeMarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    trails.NullTime
		expected []byte
	}{} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.input.MarshalJSON()
			require.Nil(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestNullTimeUnmarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    []byte
		expected *trails.NullTime
	}{
		{"nil", []byte(nil), new(trails.NullTime)},
		{"null", []byte("null"), new(trails.NullTime)},
		{"invalid", []byte("1970-01-01T00:00:00Z00:00"), new(trails.NullTime)},
		{"bad-format", []byte(`"01-01-1970T00:00:00Z00:00"`), new(trails.NullTime)},
		{
			"valid",
			[]byte(`"1970-01-01T00:00:00Z"`),
			&trails.NullTime{NullTime: sql.NullTime{Valid: true, Time: time.Unix(0, 0).UTC()}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := new(trails.NullTime)
			err := actual.UnmarshalJSON(tc.input)
			require.Nil(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
