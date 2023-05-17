package template

import (
	"fmt"
	html "html/template"
	"io/fs"
	"path"
)

// Parse implements Parser with a focus on utilizing embedded HTML templates through fs.FS.
type Parser struct {
	cache mergeFS
	fns   html.FuncMap
}

// NewParser constructs a Parse with the fses and opts.
// The order of fs.FS in fses matters.
// The first reference to a filepath,
// starting at the beginning of fses, is cached.
func NewParser(fses []fs.FS) *Parser {
	return &Parser{
		fns:   make(html.FuncMap),
		cache: merge(fses),
	}
}

func (p *Parser) clone() *Parser {
	newP := &Parser{cache: make(mergeFS), fns: make(html.FuncMap)}
	for k, v := range p.cache {
		newP.cache[k] = v
	}

	for k, v := range p.fns {
		newP.fns[k] = v
	}

	return newP
}

// Parse parses files found in the *Parse.fs with those functions provided previously.
func (p *Parser) Parse(fps ...string) (*html.Template, error) {
	var n int
	dupes := make(map[string]bool)
	for _, fp := range fps {
		if fp != "" && !dupes[fp] {
			fps[n] = fp
			dupes[fp] = true
			n++
		}
	}

	fps = fps[:n]

	if len(fps) == 0 {
		return nil, fmt.Errorf("%w", ErrNoFiles)
	}

	return html.New(path.Base(fps[0])).Funcs(p.fns).ParseFS(p.cache, fps...)
}
