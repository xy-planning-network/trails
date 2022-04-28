package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
)

const (
	assetsPath       = "/assets/"
	assetsPublicPath = "client/public/"
	clientDistPath   = "client/dist/"
)

// A Route maps a path and HTTP method to an http.HandlerFunc.
// Additional middleware.Adapters can be called when a server handles
// a request matching the Route.
type Route struct {
	Path        string
	Method      string
	Handler     http.HandlerFunc
	Middlewares []middleware.Adapter
}

// A Router handles many Routes, directing HTTP requests to the appropriate endpoint.
type Router interface {
	// AuthedRoutes registers the set of Routes as those requiring authentication.
	AuthedRoutes(key keyring.Keyable, loginUrl string, logoffUrl string, routes []Route, middlewares ...middleware.Adapter)

	// Handle applies the Route to the Router
	Handle(route Route)

	// HandleNotFound sets the provided http.HandlerFunc as the default function
	// for when no other registered Route is matched.
	HandleNotFound(handler http.HandlerFunc)

	// HandleRoutes registers the set of Routes.
	// HandleRoutes calls the provided middlewares before sending a request to the Route.
	HandleRoutes(routes []Route, middlewares ...middleware.Adapter)

	// OnEveryRequest sets the middleware stack to be applied before every request
	//
	// Other methods applying a set of middleware.Adapters will always apply theirs
	// after the set defined by OnEveryRequest.
	OnEveryRequest(middlewares ...middleware.Adapter)

	// Subrouter prefixes a Router's handling with the provided string
	Subrouter(prefix string) Router

	// UnauthedRoutes handles the set of Routes
	UnauthedRoutes(key keyring.Keyable, routes []Route, middlewares ...middleware.Adapter)

	http.Handler
}

// The DefaultRouter handles HTTP requests to any Routes it is configured with.
//
// DefaultRouter applies the middleware.ReportPanic handler to all registered routes.
//
// DefaultRouter routes requests for assets to their location in a standard trails app layout.
// DefaultRouter applies a "Cache-Control" header to responses for assets.
type DefaultRouter struct {
	Env           string
	everyReqStack []middleware.Adapter
	*mux.Router
}

// AuthedRoutes registers the set of Routes as those requiring authentication.
func (r *DefaultRouter) AuthedRoutes(
	key keyring.Keyable,
	loginUrl string,
	logoffUrl string,
	routes []Route,
	middlewares ...middleware.Adapter,
) {
	r.HandleRoutes(routes, append(middlewares, middleware.RequireAuthed(key, loginUrl, logoffUrl))...)
}

// NewRouter constructs an implementation of Router using DefaultRouter for the given environment.
//
// TODO(dlk): use provided fs.FS and http.FS instead of http.FileServer.
func NewRouter(env string) Router {
	r := mux.NewRouter()

	// NOTE(dlk): direct reqs for the client to its distribution
	r.PathPrefix("/" + clientDistPath).Handler(
		http.StripPrefix("/"+clientDistPath,
			cacheControlWrapper(http.FileServer(http.Dir(clientDistPath)))),
	)

	// NOTE(dlk): direct reqs for assets to public path
	r.PathPrefix(assetsPath).Handler(
		http.StripPrefix(assetsPath,
			cacheControlWrapper(http.FileServer(http.Dir(assetsPublicPath)))),
	)

	return &DefaultRouter{Env: env, Router: r}
}

// Handle applies the Route to the Router, wrapping the Handler in middleware.ReportPanic.
func (r *DefaultRouter) Handle(route Route) {
	r.HandleRoutes([]Route{route})
}

// HandleNotFound sets the provided http.HandlerFunc as the default function
// for when no other registered Route is matched.
func (r *DefaultRouter) HandleNotFound(handler http.HandlerFunc) {
	r.Router.NotFoundHandler = middleware.ReportPanic(r.Env)(handler)
}

// HandleRoutes registers the set of Routes on the Router and includes all the middleware.Adapters on each Route.
// Any middleware.Adapters already assigned to a Route are appended to middlewares,
// so are called after the default set.
func (r *DefaultRouter) HandleRoutes(routes []Route, middlewares ...middleware.Adapter) {
	for _, route := range routes {
		mws := append(middlewares, route.Middlewares...)
		r.Router.
			Handle(
				route.Path,
				middleware.Chain(
					middleware.ReportPanic(r.Env)(route.Handler),
					append(r.everyReqStack, mws...)...,
				),
			).
			Methods(route.Method)
	}

}

// OnEveryRequest appends the middlewares to the existing stack
// that the DefaultRouter will apply to every request.
func (r *DefaultRouter) OnEveryRequest(middlewares ...middleware.Adapter) {
	r.everyReqStack = append(r.everyReqStack, middlewares...)
}

// ServeHTTP responds to an HTTP request.
func (r *DefaultRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(w, req)
}

// Subrouter constructs a Router that handles requests to endpoints matching the prefix.
//
// e.g., r.Subrouter("/api/v1") handles requests to endpoints like /api/v1/users
func (r *DefaultRouter) Subrouter(prefix string) Router {
	return &DefaultRouter{
		Env:           r.Env,
		Router:        r.Router.PathPrefix(prefix).Subrouter(),
		everyReqStack: r.everyReqStack,
	}
}

// UnauthedRoutes registers the set of Routes as those requiring unauthenticated users.
func (r *DefaultRouter) UnauthedRoutes(key keyring.Keyable, routes []Route, middlewares ...middleware.Adapter) {
	r.HandleRoutes(routes, append(middlewares, middleware.RequireUnauthed(key))...)
}

// cacheControlWrapper helps by adding a "Cache-Control" header to the response.
func cacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		h.ServeHTTP(w, r)
	})
}
