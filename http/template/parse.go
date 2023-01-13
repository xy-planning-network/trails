package template

import (
	"fmt"
	html "html/template"
	"io/fs"
	"os"
	"path"
	"sync"
)

// Parser is the interface for parsing HTML templates with the functions provided.
type Parser interface {
	AddFn(name string, fn any)
	Parse(fps ...string) (*html.Template, error)
}

// Parse implements Parser with a focus on utilizing embedded HTML templates through fs.FS.
type Parse struct {
	fs  fs.FS
	fns html.FuncMap
}

// NewParser constructs a Parse with the provided functional options.
func NewParser(opts ...ParserOptFn) Parser {
	p := &Parse{fns: make(html.FuncMap)}
	for _, opt := range opts {
		opt(p)
	}

	userFS := p.fs
	if userFS == nil {
		userFS = os.DirFS(".")
	}

	p.fs = &mergeFS{
		cache:   make(map[string]func(string) (fs.File, error)),
		userDir: userFS,
		pkgDir:  pkgFS,
		Mutex:   sync.Mutex{},
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

	return html.New(path.Base(fps[0])).Funcs(p.fns).ParseFS(p.fs, fps...)
}
