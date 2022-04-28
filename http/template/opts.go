package template

import "io/fs"

// The ParserOptFn applies functional options to a *Parse when constructing it.
type ParserOptFn func(*Parse)

// WithFn encloses a named function so it can be added to a *Parse's function map.
func WithFn(name string, fn any) ParserOptFn {
	return func(p *Parse) {
		p.AddFn(name, fn)
	}
}

func WithFS(filesys fs.FS) ParserOptFn {
	return func(p *Parse) {
		p.fs = filesys
	}
}
