package template

import (
	"fmt"
	html "html/template"
	"io/fs"
	"path"
)

// Parser is the interface for parsing HTML templates with the functions provided.
type Parser interface {
	AddFn(name string, fn any)
	Parse(fps ...string) (*html.Template, error)
}

// Parse implements Parser with a focus on utilizing embedded HTML templates through fs.FS.
type Parse struct {
	cache mergeFS
	fns   html.FuncMap
}

// NewParser constructs a Parse with the fses and opts.
// Overwrite the embedded html templates this package provides
// by creating a filesystem (whether embedded or present in the OS)
// whose structure matches the exact filepaths (starting with tmpl/)
// for the templates you wish to overwrite.
func NewParser(fses []fs.FS, opts ...ParserOptFn) Parser {
	p := &Parse{fns: make(html.FuncMap)}
	for _, opt := range opts {
		opt(p)
	}

	p.cache = merge(fses)

	return p
}

// Parse parses files found in the *Parse.fs with those functions provided previously.
func (p *Parse) Parse(fps ...string) (*html.Template, error) {
	for i, fp := range fps {
		if fp == "" {
			fps = append(fps[:i], fps[i+1:]...)
		}
	}

	if len(fps) == 0 {
		return nil, fmt.Errorf("%w", ErrNoFiles)
	}

	return html.New(path.Base(fps[0])).Funcs(p.fns).ParseFS(p.cache, fps...)
}
