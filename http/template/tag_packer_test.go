package template_test

import (
	"fmt"
	html "html/template"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/template"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
)

const (
	cssAsset = "http://localhost:8080/css/%s.css"
	jsAsset  = "http://localhost:8080/js/%s.js"
	cssTag   = `<link rel="stylesheet" href="%s">`
	jsTag    = `<script src="%s" type="text/javascript"></script>`
	cssGlob  = "client/dist/css/%s.*.css"
	jsGlob   = "client/dist/js/%s.*.js"
)

func TestTagPacker(t *testing.T) {
	// Arrange
	tcs := []struct {
		name     string
		filename string
		isCSS    bool
		fn       func(string, bool) html.HTML
		expected html.HTML
	}{
		{"env-testing", "", false, template.TagPacker("testing", nil), html.HTML("")},
		{"env-case-matching", "", false, template.TagPacker("TeStiNG", nil), html.HTML("")},
		{
			"env-dev-zero-name-js",
			"",
			false,
			template.TagPacker("development", nil),
			html.HTML(fmt.Sprintf(jsTag, fmt.Sprintf(jsAsset, ""))),
		},
		{
			"env-dev-js",
			"test",
			false,
			template.TagPacker("development", nil),
			html.HTML(fmt.Sprintf(jsTag, fmt.Sprintf(jsAsset, "test"))),
		},
		{
			"env-dev-css",
			"test",
			true,
			template.TagPacker("development", nil),
			html.HTML(fmt.Sprintf(cssTag, fmt.Sprintf(cssAsset, "test"))),
		},
		{
			"env-prod-glob-not-found",
			"test",
			false,
			template.TagPacker("production", tt.NewMockFS(tt.NewMockFile("some/other/js/test.js", nil))),
			html.HTML(fmt.Sprintf(jsTag, "error-not-found")),
		},
		{
			"env-prod-js",
			"test",
			false,
			template.TagPacker(
				"production",
				tt.NewMockFS(tt.NewMockFile("client/dist/js/test.file.js", nil)),
			),
			html.HTML(fmt.Sprintf(jsTag, "/client/dist/js/test.file.js")),
		},
		{
			"env-prod-css",
			"test",
			true,
			template.TagPacker(
				"production",
				tt.NewMockFS(tt.NewMockFile("client/dist/css/test.file.css", nil)),
			),
			html.HTML(fmt.Sprintf(cssTag, "/client/dist/css/test.file.css")),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual := tc.fn(tc.filename, tc.isCSS)

			// Assert
			require.Equal(t, tc.expected, actual)
		})
	}
}
