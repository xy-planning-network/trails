/*

Package main provides a toy example use of Trails' http stack.

*/
package main

import (
	"embed"
	"errors"
	"fmt"
	html "html/template"
	"net/http"
	"time"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/ranger"
)

//go:embed *.tmpl nested/*.tmpl tmpl/layout/*.tmpl
var files embed.FS

type ctxKey string

func (k ctxKey) Key() string { return string(k) }
func (k ctxKey) String() string {
	return "http/example context key: " + string(k)
}

const (
	sessionKey ctxKey = "example-session-key"
	userKey    ctxKey = "example-user-key"

	// these refer to templates that should be available for rendering
	dir     string = ""
	base    string = dir + "base.tmpl"
	auth    string = dir + "auth.tmpl"
	content string = dir + "content.tmpl"
	first   string = dir + "main.tmpl"
	nav     string = dir + "nav.tmpl"
	other   string = dir + "nested/other.tmpl"
)

// Example template function to inject.
func itsHammerTime() int64 { return time.Now().UnixNano() }

// Handler shares the initialized Responder across all example responses.
type Handler struct {
	*Responder
	Ring keyring.Keyringable
}

type RangerHandler struct {
	*ranger.Ranger
}

// root is a fully-formed use of Responder.
func (h *RangerHandler) root(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
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
// in this example we show how values provided in one response do not bleed into another.
// to test this out, throw in different values into a query param: ?name=
// then, remove it from your request to see the name resets.
func (h *RangerHandler) withCurrentUser(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(h.Ranger.Keyring.CurrentUserKey()).(middleware.User)
	if !ok {
		err := errors.New("no user")
		h.Redirect(w, r, Err(err), ToRoot())
		return
	}

	if err := h.Html(w, r, Authed(), Tmpls(first)); err != nil {
		h.Err(w, r, err)
	}
}

// incorrect shows how a template not actually referred to by the base does
// not break our ability to call Html.
func (h *RangerHandler) incorrect(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Unauthed(), Tmpls(other, nav, content)); err != nil {
		h.Err(w, r, err)
	}
}

// broken cannot render because no authed template was set on the Responder.
func (h *RangerHandler) broken(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Authed()); err != nil {
		h.Err(w, r, err)
	}
}

type lo string

func (l lo) Debug(_ string, _ *logger.LogContext) { fmt.Println(l) }
func (l lo) Info(_ string, _ *logger.LogContext)  { fmt.Println(l) }
func (l lo) Warn(_ string, _ *logger.LogContext)  { fmt.Println(l) }
func (l lo) Error(_ string, _ *logger.LogContext) { fmt.Println(l) }
func (l lo) Fatal(_ string, _ *logger.LogContext) { fmt.Println(l) }
func (lo) LogLevel() logger.LogLevel              { return 99999999 }

type poopoo string

func (poopoo) AddFn(_ string, _ interface{}) {}
func (p poopoo) Parse(_ ...string) (*html.Template, error) {
	return html.Must(html.New("unauthenticated_base.tmpl").Parse(string(p))), nil
}

var (
	_ logger.Logger   = lo("hey")
	_ template.Parser = poopoo("hey")
)

func main() {
	p := ranger.DefaultParser(files, template.WithFn("hammer", itsHammerTime))
	rng, err := ranger.NewRanger(p)
	if err != nil {
		fmt.Println(err)
		return
	}

	h := &RangerHandler{Ranger: rng}
	rng.Router.HandleRoutes(
		[]router.Route{
			{Path: "/broken", Method: http.MethodGet, Handler: h.broken},
			{Path: "/incorrect", Method: http.MethodGet, Handler: h.incorrect},
			{Path: "/with-user", Method: http.MethodGet, Handler: h.withCurrentUser},
			{Path: "/", Method: http.MethodGet, Handler: h.root},
		},
	)
	rng.Router.HandleNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not found")) }))

	if err := rng.Guide(); err != nil {
		fmt.Println(err)
		return
	}
	/*
		log := logger.NewLogger()
		u, err := url.ParseRequestURI("localhost:8081")
		if err != nil {
			log.Fatal(err.Error(), nil)
			os.Exit(1)
		}

		// Setup new keyring.Keyring
		ring := keyring.NewKeyring(sessionKey, userKey)

		// Setup new template.Parser
		//
		// notably, many of the functions passed in are closures
		// we thereby make available to all handlers values that are like constants/globals
		p := template.NewParser(
			files,
			template.WithFn("hammer", itsHammerTime),
			template.WithFn(template.Env(env)),
			template.WithFn(template.RootUrl(u)),
		)

		// Setup new resp.Responder
		d := NewResponder(
			WithSessionKey(ring.SessionKey()),
			WithUserSessionKey(ring.CurrentUserKey()),
			WithRootUrl("/"),
			WithParser(p),
			WithAuthTemplate(auth),
			WithUnauthTemplate(base),
		)

		// Setup session store
		sessionstore, err := session.NewStoreService(env, "f0f42f970982be947b6536df8a0d2489d7f06d15e6d36cfbfae7ac16ccbe7cc59780b3fac338e4e0d3c5968b1fa24a884932547614ce16fdb1a18ba089f44af0", "0bc7730e33bb9ef91197239cb3a44fb43a4f3fac7b36e36afd4f2b2d8e920b65")
		if err != nil {
			log.Fatal(err.Error(), nil)
			os.Exit(1)
		}

		// Setup user store
		userstore := make(users, 0)

		// Setup visitor tracker
		vs := middleware.NewVisitors()

		// Setup handler manager
		h := &Handler{Responder: d, Ring: ring}

		// Setup router
		r := router.NewRouter(env)
		r.HandleNotFound(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("Oops no " + r.URL.Path))
		})

		// Include the following default middleware stack, executing on every HTTP request.
		r.OnEveryRequest(
			middleware.RateLimit(vs),
			middleware.LogRequest(log),
			middleware.InjectIPAddress(),
			middleware.InjectSession(sessionstore, ring.SessionKey()),
			registerUser(ring.SessionKey(), &userstore),
			middleware.CurrentUser(d, &userstore, ring.SessionKey(), ring.CurrentUserKey()),
		)

		r.HandleRoutes(
			[]router.Route{
				{Path: "/broken", Method: http.MethodGet, Handler: h.broken},
				{Path: "/incorrect", Method: http.MethodGet, Handler: h.incorrect},
				{Path: "/with-user", Method: http.MethodGet, Handler: h.withCurrentUser},
				{Path: "/", Method: http.MethodGet, Handler: h.root},
			},
		)

		// Run the web server
		http.ListenAndServe(u.String(), r)
	*/
}
