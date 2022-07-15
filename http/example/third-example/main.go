/*

third-example provides a more "robust" use of authentication and unauthentication,
highlighting trails' flexibility to work
with an application's own implementation of important interfaces
describing the currentUser concept at the heart of trails.

As well, it leverages middleware.InjectResponder instead of wrapping the Ranger's Responder
in some intermediate struct HTTP handlers are methods on,
thereby allowing those handlers to be standalone functions.

*/
package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/ranger"
)

const responderCtxKey keyring.Key = "third-example-responder-ctx-key"

var (
	sessionKey keyring.Keyable
)

// mockUser is our custom "user" type, albeit an overly simple one.
// Most apps will not use
type mockUser uint

func (u mockUser) HasAccess() bool  { return true }
func (u mockUser) HomePath() string { return "/home" }

// Our mockUserStorer shows how our user type - mockUser custom implementation of middleware.User
// need not use trails.User.
type mockUserStorer struct{}

func (u mockUserStorer) GetByID(id uint) (middleware.User, error) {
	return mockUser(id), nil
}

func home(w http.ResponseWriter, r *http.Request) {
	rp, ok := r.Context().Value(responderCtxKey).(*Responder)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := rp.Html(w, r, Authed(), Tmpls("home.tmpl")); err != nil {
		rp.Err(w, r, err)
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	rp, ok := r.Context().Value(responderCtxKey).(*Responder)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		err = fmt.Errorf("bad id query param %q", r.URL.Query().Get("id"))
		rp.Redirect(w, r, Err(err))

		return
	}

	sess, ok := r.Context().Value(sessionKey).(session.TrailsSessionable)
	if !ok {
		err := errors.New("expected session in *http.Request.Context, not found")
		rp.Err(w, r, err)

		return
	}

	if err := sess.RegisterUser(w, r, uint(id)); err != nil {
		rp.Err(w, r, err)

		return
	}

	rp.Redirect(w, r, Url("/"))
}

func logoff(w http.ResponseWriter, r *http.Request) {
	rp, ok := r.Context().Value(responderCtxKey).(*Responder)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sess, ok := r.Context().Value(sessionKey).(session.TrailsSessionable)
	if !ok {
		err := errors.New("expected session in *http.Request.Context, not found")
		rp.Err(w, r, err)

		return
	}

	if err := sess.DeregisterUser(w, r); err != nil {
		rp.Err(w, r, err)
		return
	}

	rp.Redirect(w, r)
}

func root(w http.ResponseWriter, r *http.Request) {
	rp, ok := r.Context().Value(responderCtxKey).(*Responder)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := rp.Html(w, r, Unauthed(), Tmpls("root.tmpl")); err != nil {
		rp.Err(w, r, err)
		return
	}
}

func main() {
	rng, err := ranger.New(
		ranger.WithUserSessions(mockUserStorer{}),
		ranger.DefaultResponder(
			WithAuthTemplate("auth.tmpl"),
			WithUnauthTemplate("unauth.tmpl"),
		),
	)
	if err != nil {
		return
	}

	sessionKey = rng.EmitKeyring().SessionKey()

	rng.OnEveryRequest(middleware.InjectResponder(rng.Responder, responderCtxKey))
	rng.UnauthedRoutes(
		rng.EmitKeyring().CurrentUserKey(),
		[]router.Route{
			{Path: "/login", Method: http.MethodGet, Handler: login},
			{Path: "/", Method: http.MethodGet, Handler: root},
		},
	)

	rng.AuthedRoutes(
		rng.EmitKeyring().CurrentUserKey(),
		"/login",
		"/logoff",
		[]router.Route{
			{Path: "/home", Method: http.MethodGet, Handler: home},
			{Path: "/logoff", Method: http.MethodGet, Handler: logoff},
		},
	)

	if err := rng.Guide(); err != nil {
		rng.Error(err.Error(), nil)
		return
	}
}
