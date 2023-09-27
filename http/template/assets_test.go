package template_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/template"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
)

const (
	assetURI = "%sclient/dist/%s"
	origin   = "https://cdn.xypn.com"
)

func TestAssetURI(t *testing.T) {
	originURL, _ := url.Parse(origin)

	// Arrange
	tcs := []struct {
		name     string
		filepath string
		fn       func(string) string
		expected string
	}{
		{"env-testing", "", template.AssetURI(nil, trails.Testing, nil, nil), ""},
		{
			"zero-name",
			"",
			template.AssetURI(nil, trails.Development, nil, nil),
			fmt.Sprintf(assetURI, "/", ""),
		},
		{
			"no-hash-match-no-origin",
			"assets/test.js",
			template.AssetURI(nil, trails.Production, nil, nil),
			fmt.Sprintf(assetURI, "/", "assets/test.js"),
		},
		{
			"hash-match-no-origin",
			"assets/test.js",
			template.AssetURI(nil, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil)), nil),
			fmt.Sprintf(assetURI, "/", "assets/test-af8s7f9.js"),
		},
		{
			"hash-match-no-origin",
			"assets/test.js",
			template.AssetURI(nil, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil)), nil),
			fmt.Sprintf(assetURI, "/", "assets/test-af8s7f9.js"),
		},
		{
			"hash-match-origin",
			"assets/test.js",
			template.AssetURI(originURL, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil)), nil),
			fmt.Sprintf(assetURI, "https://cdn.xypn.com/", "assets/test-af8s7f9.js"),
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
