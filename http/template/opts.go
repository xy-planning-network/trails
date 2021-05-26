package template

type ParserOptFn func(*Parse)

func WithFn(name string, fn interface{}) ParserOptFn {
	return func(p *Parse) {
		p.fns[name] = fn
	}
}
