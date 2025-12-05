package trails_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

func TestMask(t *testing.T) {
	for _, tc := range []struct {
		name string
		vals url.Values
		key  string
		want url.Values
	}{
		{"zero", url.Values{}, "", url.Values{}},
		{
			"mismatch",
			url.Values{"password": []string{"hunter2"}},
			"passwrod",
			url.Values{"password": []string{"hunter2"}},
		},
		{
			"match",
			url.Values{"password": []string{"hunter2"}},
			"password",
			url.Values{"password": []string{trails.LogMaskVal}},
		},
		{
			"squash-multiple",
			url.Values{"password": []string{"hunter2", "hunter3"}},
			"password",
			url.Values{"password": []string{trails.LogMaskVal}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trails.Mask(tc.vals, tc.key)
			require.Equal(t, tc.want, tc.vals)
		})
	}
}
