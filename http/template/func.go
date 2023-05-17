package template

import (
	html "html/template"
	"net/url"

	"github.com/google/uuid"
	"github.com/xy-planning-network/trails"
)

// AddFn includes the named function in the Parse function map.
func (p *Parser) AddFn(name string, fn any) *Parser {
	newP := p.clone()
	if newP.fns == nil {
		newP.fns = make(html.FuncMap)
	}

	newP.fns[name] = fn

	return newP
}

// CurrentUser encloses some value representing a user.
// It returns "currentUser" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning the enclosed value when called.
func CurrentUser(u any) (string, func() any) {
	return "currentUser", func() any { return u }
}

// Env encloses some string representing an environment.
// It returns "env" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning the enclosed value when called.
func Env(e trails.Environment) (string, func() string) {
	return "env", func() string { return e.String() }
}

// Nonce returns "nonce" as the name of the function for convenient passing to a template.FuncMap
// and returns a function generating a uuid.
func Nonce() (string, func() string) {
	return "nonce", func() string { return uuid.NewString() }
}

// RootUrl encloses the *url.URL representing the base URL of the web app.
// It returns "rootUrl" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning its *url.URL.String().
// If u is nil, that function will always return an empty string.
func RootUrl(u *url.URL) (string, func() string) {
	if u == nil {
		return "rootUrl", func() string { return "" }
	}

	s := u.String()
	return "rootUrl", func() string { return s }
}
