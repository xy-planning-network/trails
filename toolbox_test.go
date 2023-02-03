package trails_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

func TestToolboxFilter(t *testing.T) {
	// TODO
	toolbox := make(trails.Toolbox, 1)
	require.Len(t, toolbox.Filter(), 0)
}

func TestToolRender(t *testing.T) {
	// TODO
	var tool trails.Tool
	require.False(t, tool.Render())
}
