package template

import (
	"errors"
	"fmt"
	html "html/template"
	"io/fs"
	"os"
	"path"

	"github.com/xy-planning-network/trails"
)

const (
	cssGlob = "client/dist/css/%s.*.css"
	jsGlob  = "client/dist/js/%s.*.js"
)

// TagPacker encloses the environment and filesystem so when called executing a template,
// emits valid paths to JS and CSS assets.
//
// TODO(dlk):
//
// - configurable asset paths?
// - represent error states; ship files in trails generate?
func TagPacker(env trails.Environment, filesys fs.FS) func(string, bool) html.HTML {
	if filesys == nil {
		filesys = os.DirFS(".")
	}
	return func(name string, isCSS bool) html.HTML {
		assetPath := fmt.Sprintf("http://localhost:8080/js/%s.js", name)
		if env.IsDevelopment() {
			assetPath = fmt.Sprintf("http://localhost:8080/src/pages/%s.ts", name)
		}
		tagTemplate := `<script src="%s" type="module"></script>`
		glob := fmt.Sprintf(jsGlob, name)

		if isCSS {
			assetPath = fmt.Sprintf("http://localhost:8080/css/%s.css", name)
			tagTemplate = `<link rel="stylesheet" href="%s">`
			glob = fmt.Sprintf(cssGlob, name)
		}

		switch {
		case env.IsTesting():
			return ""

		case env.IsDevelopment():
			return html.HTML(fmt.Sprintf(tagTemplate, assetPath))

		default:
			matches, err := fs.Glob(filesys, glob)
			if errors.Is(err, path.ErrBadPattern) {
				return html.HTML(fmt.Sprintf(tagTemplate, "error-bad-glob"))
			}
			if len(matches) == 0 {
				return html.HTML(fmt.Sprintf(tagTemplate, "error-not-found"))
			}
			return html.HTML(fmt.Sprintf(tagTemplate, "/"+matches[0]))
		}
	}
}
