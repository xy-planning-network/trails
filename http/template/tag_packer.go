package template

import (
	"errors"
	"fmt"
	html "html/template"
	"io/fs"
	"os"
	"path"
	"strings"
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
func TagPacker(env string, filesys fs.FS) func(string, bool) html.HTML {
	if filesys == nil {
		filesys = os.DirFS(".")
	}
	return func(name string, isCSS bool) html.HTML {
		assetPath := fmt.Sprintf("http://localhost:8080/js/%s.js", name)
		tagTemplate := `<script src="%s" type="text/javascript"></script>`
		glob := fmt.Sprintf(jsGlob, name)

		if isCSS {
			assetPath = fmt.Sprintf("http://localhost:8080/css/%s.css", name)
			tagTemplate = `<link rel="stylesheet" href="%s">`
			glob = fmt.Sprintf(cssGlob, name)
		}

		// TODO(dlk): use domain.Environment
		switch {
		case strings.EqualFold("testing", env):
			return ""

		case strings.EqualFold("development", env):
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
