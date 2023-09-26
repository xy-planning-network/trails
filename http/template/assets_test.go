package template_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/template"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
)

const (
	devAsset  = "http://localhost:8080/client/dist/%s"
	prodAsset = "/client/dist/%s"
)

func TestAssetURI(t *testing.T) {
	// Arrange
	tcs := []struct {
		name     string
		filepath string
		fn       func(string) string
		expected string
	}{
		{"env-testing", "", template.AssetURI(trails.Testing, nil), ""},
		{
			"env-dev-zero-name-js",
			"",
			template.AssetURI(trails.Development, nil),
			fmt.Sprintf(devAsset, ""),
		},
		{
			"env-dev",
			"src/pages/test.ts",
			template.AssetURI(trails.Development, nil),
			fmt.Sprintf(devAsset, "src/pages/test.ts"),
		},
		{
			"env-prod-no-match",
			"assets/test.js",
			template.AssetURI(trails.Production, nil),
			fmt.Sprintf(prodAsset, "assets/test.js"),
		},
		{
			"env-prod",
			"assets/test.js",
			template.AssetURI(trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil))),
			fmt.Sprintf(prodAsset, "assets/test-af8s7f9.js"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual := tc.fn(tc.filepath)

			// Assert
			require.Equal(t, tc.expected, actual)
		})
	}
}
