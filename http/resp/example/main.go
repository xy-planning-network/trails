package main

import (
	"context"
	"embed"
	"net/http"
	"net/url"
	"time"

	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
)

//go:embed *.tmpl nested/*.tmpl
var files embed.FS

type ctxKey string

func (k ctxKey) Key() string    { return string(k) }
func (k ctxKey) String() string { return string(k) }

const (
	key ctxKey = "key"

	// these refer to templates that should be available for rendering
	base    string = "base.tmpl"
	auth    string = "auth.tmpl"
	content string = "content.tmpl"
	first   string = "main.tmpl"
	nav     string = "nav.tmpl"
	other   string = "nested/other.tmpl"
)

// Example template function to inject.
func itsHammerTime() int64 { return time.Now().UnixNano() }

// Handler shares the initialized Responder across all example responses.
type Handler struct {
	*Responder
}

// root is a fully-formed use of Responder.
func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"sick": "such data",
		"wow":  "so data",
		"ooh":  "dataaaa",
	}
	if err := h.Html(w, r, Unauthed(), Tmpls(first, nav, content), Data(data)); err != nil {
		h.Err(w, r, err)
	}
}

// withCurrentUser is a fully-formed use of Responder passing data from resp opts to template rendering.
//
// in this example we show how a values provided in one response do not bleed into another.
// to test this out, throw in differnt values into a query param: ?name=
// then, remove it from your request to see the name resets.
func (h *Handler) withCurrentUser(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("name")
	if err := h.Html(w, r, Authed(), Tmpls(first), User(struct{ Name string }{u})); err != nil {
		h.Err(w, r, err)
	}
}

// incorrect shows how a template not actually referred to by the base does
// not break our ability to call Render.
func (h *Handler) incorrect(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Unauthed(), Tmpls(other, nav, content)); err != nil {
		h.Err(w, r, err)
	}
}

// broken cannot render because no authed template was set on the Responder.
func (h *Handler) broken(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Authed()); err != nil {
		h.Err(w, r, err)
	}
}

// we need sessions in order to use Responder, so let's use a stubbed one.
func injectSession(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), key, session.Stub{})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func main() {
	u, _ := url.ParseRequestURI("localhost:8081")

	// allocate our parser
	//
	// notably, many of the functions passed in are closures
	// we make available to all handlers values that are like constants/globals
	p := template.NewParser(
		files,
		template.WithFn("hammer", itsHammerTime),
		template.WithFn(template.Env("EXAMPLE")),
		template.WithFn(template.RootUrl(u)),
	)

	// allocate our responder
	d := NewResponder(WithSessionKey(key), WithParser(p), WithAuthTemplate(auth), WithUnauthTemplate(base))

	// setup routing and middleware
	h := &Handler{d}
	http.HandleFunc("/broken", injectSession(h.broken))
	http.HandleFunc("/incorrect", injectSession(h.incorrect))
	http.HandleFunc("/with-user", injectSession(h.withCurrentUser))
	http.HandleFunc("/", injectSession(h.root))

	// run the server
	http.ListenAndServe(u.String(), nil)
}
