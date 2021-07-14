package template

import (
	"fmt"
	html "html/template"
	"io/fs"
)

// Parser is the interface for parsing HTML templates with the functions provided.
type Parser interface {
	AddFn(name string, fn interface{})
	Parse(fps ...string) (*html.Template, error)
}

// Parse implements Parser with a focus on utilizing embedded HTML templates through fs.FS.
type Parse struct {
	fs  fs.FS
	fns html.FuncMap
}

// NewParser constructs a Parse with the provided fs.FS and functional options.
func NewParser(fs fs.FS, opts ...ParserOptFn) Parser {
	p := &Parse{fs: fs, fns: make(html.FuncMap)}
	for _, opt := range opts {
		opt(p)
	}
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

	return html.New(fps[len(fps)-1]).Funcs(p.fns).ParseFS(p.fs, fps...)
}
