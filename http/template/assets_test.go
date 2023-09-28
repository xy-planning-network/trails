package template_test

import (
	"fmt"
	"net/url"
	"path"
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
		name        string
		filepath    string
		fn          func(string) (string, error)
		expected    string
		expectedErr error
	}{
		{"env-testing", "", template.AssetURI(nil, trails.Testing, nil), "", nil},
		{
			"zero-name",
			"",
			template.AssetURI(nil, trails.Development, nil),
			fmt.Sprintf(assetURI, "/", ""),
			nil,
		},
		{
			"no-hash-match-no-origin",
			"assets/test.js",
			template.AssetURI(nil, trails.Production, nil),
			fmt.Sprintf(assetURI, "/", "assets/test.js"),
			nil,
		},
		{
			"no-hash-match-with-origin",
			"assets/test.js",
			template.AssetURI(originURL, trails.Production, nil),
			fmt.Sprintf(assetURI, "https://cdn.xypn.com/", "assets/test.js"),
			nil,
		},
		{
			"hash-match-no-origin",
			"assets/test.js",
			template.AssetURI(nil, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil))),
			fmt.Sprintf(assetURI, "/", "assets/test-af8s7f9.js"),
			nil,
		},
		{
			"hash-match-with-origin",
			"assets/test.js",
			template.AssetURI(originURL, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil))),
			fmt.Sprintf(assetURI, "https://cdn.xypn.com/", "assets/test-af8s7f9.js"),
			nil,
		},
		{
			"err-multiple-matches",
			"assets/test.js",
			template.AssetURI(originURL, trails.Production, tt.NewMockFS(tt.NewMockFile("client/dist/assets/test-af8s7f9.js", nil), tt.NewMockFile("client/dist/assets/test-989fa99.js", nil))),
			"",
			template.ErrMatchedAssets,
		},
		{
			"err-bad-pattern",
			"assets/test.js[",
			template.AssetURI(originURL, trails.Production, nil),
			"",
			path.ErrBadPattern,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual, err := tc.fn(tc.filepath)

			// Assert
			require.Equal(t, tc.expected, actual)
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
