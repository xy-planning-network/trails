/*

start-here provides a toy example use of Trails' http stack,
focusing on the basics of:

(1) constructing a default Ranger;
(2) binding routes to handlers;
(3) using resp.Responder methods for responding to requests;
(4) and the use of resp.Fn functional options for declaring how
	the method forms the response payload.
*/
package main

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/ranger"
)

const (
	// these refer to templates available for rendering
	dir    string = "tmpl/"
	last   string = dir + "last.tmpl"
	first  string = dir + "first.tmpl"
	next   string = dir + "next.tmpl"
	nested string = dir + "nested/nested.tmpl"
)

// RangerHandler wraps a configured *Ranger.
// The methods attached to it are the handlers the Router
// will direct requests to.
type RangerHandler struct {
	*ranger.Ranger
}

// authed is a fully-formed use of Responder showing how the inclusion of a user
// in the *http.Request.Context allows using Authed(),
// in contrast to the broken method below which does not.
func (h *RangerHandler) authed(w http.ResponseWriter, r *http.Request) {
	// NOTE: this mocks the functionality of middleware.CurrentUser
	// which sets a user in the *http.Request.Context
	r = r.WithContext(context.WithValue(r.Context(), h.EmitKeyring().CurrentUserKey(), "example-user"))

	if err := h.Html(w, r, Authed(), Tmpls(first)); err != nil {
		h.Err(w, r, err)
	}
}

// root is a fully-formed use of Responder.
//
// Unauthed does not error out because an unauthenticated template
// is found at the default location by ranger: tmpl/layout/unauthenticated_base.tmpl.
func (h *RangerHandler) root(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"sick": "such data",
		"wow":  "so data",
		"ooh":  "dataaaa",
	}
	if err := h.Html(w, r, Unauthed(), Tmpls(first, next, last), Data(data)); err != nil {
		h.Err(w, r, err)
	}
}

// incorrect shows how including a template not referred to by the base (unauthenticated_base.tmpl) one
// does not break our ability to call Html.
func (h *RangerHandler) incorrect(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Unauthed(), Tmpls(nested, next, last)); err != nil {
		h.Err(w, r, err)
	}
}

// broken cannot render because there is no user to populate the authed template with.
func (h *RangerHandler) broken(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Authed()); err != nil {
		h.Err(w, r, err)
	}
}

func main() {
	//func run() {
	// construct a Ranger using all defaults.
	rng, err := ranger.New()
	if err != nil {
		fmt.Println(err)
		return
	}

	// wrap the constructed Ranger so it is exposed to all HTTP handlers.
	h := RangerHandler{rng}

	// bind routes and handlers to one another.
	// this is a group of routes that share a middleware stack.
	// in this case, no additional middleware is needed
	// beyond the default stack set for every request.
	rng.HandleRoutes(
		[]router.Route{
			{Path: "/broken", Method: http.MethodGet, Handler: h.broken},
			{Path: "/incorrect", Method: http.MethodGet, Handler: h.incorrect},
			{Path: "/authed", Method: http.MethodGet, Handler: h.authed},
			{Path: "/", Method: http.MethodGet, Handler: h.root},
		},
	)

	// start the web server until receiving a signal to stop.
	if err := rng.Guide(); err != nil {
		fmt.Println(err)
		return
	}
}
