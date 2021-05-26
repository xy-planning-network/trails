package template

import (
	"fmt"
	html "html/template"
	"io/fs"
)

type Parser interface {
	Parse(fps ...string) (*html.Template, error)
}

type Parse struct {
	fs  fs.FS
	fns html.FuncMap
}

func NewParser(fs fs.FS, opts ...ParserOptFn) Parser {
	p := &Parse{fs: fs, fns: make(html.FuncMap)}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

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
