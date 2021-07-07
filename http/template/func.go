package template

import (
	html "html/template"
	"net/url"

	uuid "github.com/satori/go.uuid"
)

// AddFn includes the named function in the Parse function map.
func (p *Parse) AddFn(name string, fn interface{}) {
	if p.fns == nil {
		p.fns = make(html.FuncMap)
	}
	p.fns[name] = fn
}

// CurrentUser encloses some value representing a user.
// It returns "currentUser" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning the enclosed value when called.
func CurrentUser(u interface{}) (string, func() interface{}) {
	return "currentUser", func() interface{} { return u }
}

// Env encloses some string representing an environment.
// It returns "env" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning the enclosed value when called.
func Env(e string) (string, func() string) {
	return "env", func() string { return e }
}

// Nonce returns "nonce" as the name of the function for convenient passing to a template.FuncMap
// and returns a function generating a uuid.
func Nonce() (string, func() string) {
	return "nonce", func() string { return uuid.NewV4().String() }
}

// RootURL encloses the *url.URL representing the base URL of the web app.
// It returns "rootURL" as the name of the function for convenient passing to a template.FuncMap
// and returns a function returning its *url.URL.String().
// If u is nil, that function will always return an empty string.
func RootURL(u *url.URL) (string, func() string) {
	if u == nil {
		return "rootURL", func() string { return "" }
	}

	s := u.String()
	return "rootURL", func() string { return s }
}
