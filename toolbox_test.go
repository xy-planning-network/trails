package trails_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

func TestToolboxFilter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  trails.Toolbox
		output trails.Toolbox
	}{
		{"Nil", nil, make(trails.Toolbox, 0)},
		{"Zero", make(trails.Toolbox, 0), make(trails.Toolbox, 0)},
		{"Filter-All", make(trails.Toolbox, 4), make(trails.Toolbox, 0)},
		{
			"From-4-To-1",
			trails.Toolbox{
				{}, {}, {},
				{Actions: make([]trails.ToolAction, 1)},
			},
			trails.Toolbox{{Actions: make([]trails.ToolAction, 1)}},
		},
		{
			"Keep-All",
			trails.Toolbox{
				{Actions: make([]trails.ToolAction, 1)},
				{Actions: make([]trails.ToolAction, 1)},
			},
			trails.Toolbox{
				{Actions: make([]trails.ToolAction, 1)},
				{Actions: make([]trails.ToolAction, 1)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, tc.input.Filter())
		})
	}
}

func TestToolRender(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  []trails.ToolAction
		output bool
	}{
		{"Nil", nil, false},
		{"Zero", make([]trails.ToolAction, 0), false},
		{"Has-Some", make([]trails.ToolAction, 3), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := trails.Tool{Actions: tc.input}
			require.Equal(t, tc.output, actual.Render())
		})
	}
}
