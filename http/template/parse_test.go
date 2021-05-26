package template_test

import (
	"bytes"
	html "html/template"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/template"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
)

type testFn func(*testing.T, *html.Template, error)

func TestParse(t *testing.T) {
	stub := []byte("<!DOCTYPE html>\n<html></html>")
	tcs := []struct {
		name   string
		parser template.Parser
		fps    []string
		assert testFn
	}{
		{
			name:   "Zero-Value",
			parser: template.NewParser(tt.NewMockFS()),
			fps:    []string{},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.ErrorIs(t, err, template.ErrNoFiles)
				require.Nil(t, tmpl)
			},
		},
		{
			name:   "Empty-String",
			parser: template.NewParser(tt.NewMockFS(tt.NewMockTmpl("", nil))),
			fps:    []string{""},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.ErrorIs(t, err, template.ErrNoFiles)
				require.Nil(t, tmpl)
			},
		},
		{
			name:   "No-File",
			parser: template.NewParser(tt.NewMockFS(tt.NewMockTmpl("", nil))),
			fps:    []string{"example.tmpl"},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.NotNil(t, err)
				require.Nil(t, tmpl)
			},
		},
		{
			name:   "Empty-File",
			parser: template.NewParser(tt.NewMockFS(tt.NewMockTmpl("example.tmpl", nil))),
			fps:    []string{"example.tmpl"},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.Nil(t, err)
				require.Equal(t, "example.tmpl", tmpl.Name())

				b := new(bytes.Buffer)
				require.Nil(t, tmpl.Execute(b, nil))
				require.Nil(t, b.Bytes())
			},
		},
		{
			name:   "Not-Empty-File",
			parser: template.NewParser(tt.NewMockFS(tt.NewMockTmpl("example.tmpl", stub))),
			fps:    []string{"example.tmpl"},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.Nil(t, err)
				require.Equal(t, "example.tmpl", tmpl.Name())

				b := new(bytes.Buffer)
				require.Nil(t, tmpl.Execute(b, nil))
				require.Equal(t, stub, b.Bytes())
			},
		},
		{
			name: "Many-Files",
			parser: template.NewParser(
				tt.NewMockFS(
					tt.NewMockTmpl(
						"example.tmpl",
						[]byte(`<!DOCTYPE html><html>{{ template "test" }}</html>`),
					),
					tt.NewMockTmpl(
						"test.tmpl",
						[]byte(`{{ define "test" }}<p>sup</p>{{ end }}`),
					),
				),
			),
			fps: []string{"example.tmpl", "test.tmpl"},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.Nil(t, err)
				require.Equal(t, "test.tmpl", tmpl.Name())

				b := new(bytes.Buffer)
				require.Nil(t, tmpl.ExecuteTemplate(b, "example.tmpl", nil))
				require.Equal(t, "<!DOCTYPE html><html><p>sup</p></html>", b.String())
			},
		},
		{
			name: "With-Fns",
			parser: template.NewParser(
				tt.NewMockFS(
					tt.NewMockTmpl(
						"example.tmpl",
						[]byte("<!DOCTYPE html><html>{{ test }}</html>"),
					),
				),
				template.WithFn("test", func() string { return "test" }),
			),
			fps: []string{"example.tmpl"},
			assert: func(t *testing.T, tmpl *html.Template, err error) {
				require.Nil(t, err)
				require.Equal(t, "example.tmpl", tmpl.Name())

				b := new(bytes.Buffer)
				require.Nil(t, tmpl.Execute(b, nil))
				require.Equal(t, "<!DOCTYPE html><html>test</html>", b.String())
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := tc.parser.Parse(tc.fps...)
			tc.assert(t, tmpl, err)
		})
	}
}

func BenchmarkParse(b *testing.B) {

}
