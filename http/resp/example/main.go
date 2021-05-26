package main

import (
	"context"
	"embed"
	"net/http"
	"time"

	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
)

//go:embed *.tmpl nested/*.tmpl
var files embed.FS

const (
	key string = "key"

	// these refer to templates that should be available for rendering
	base    string = "base.tmpl"
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
	if err := h.Render(w, r, Unauthed(), Tmpls(first, nav, content), Data(data)); err != nil {
		h.Err(w, r, err)
	}
}

// incorrect shows how a template not actually referred to by the base does
// not break our ability to call Render.
func (h *Handler) incorrect(w http.ResponseWriter, r *http.Request) {
	if err := h.Render(w, r, Unauthed(), Tmpls(other, nav, content)); err != nil {
		h.Err(w, r, err)
	}
}

// broken cannot render because no authed template was set on the Responder.
func (h *Handler) broken(w http.ResponseWriter, r *http.Request) {
	if err := h.Render(w, r, Authed()); err != nil {
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
	// allocate our parser
	p := template.NewParser(files, template.WithFn("hammer", itsHammerTime))

	// allocate our responder
	d := NewResponder(WithSessionKey(key), WithParser(p), WithUnauthTemplate(base))

	// setup routing and middleware
	h := &Handler{d}
	http.HandleFunc("/broken", injectSession(h.broken))
	http.HandleFunc("/incorrect", injectSession(h.incorrect))
	http.HandleFunc("/", injectSession(h.root))

	// run the server
	http.ListenAndServe(":8081", nil)
}
