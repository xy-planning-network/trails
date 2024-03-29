package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xy-planning-network/trails/http/middleware"
)

const (
	assetsPath = "client/dist/"
)

// A Route maps a path and HTTP method to an [http.HandlerFunc].
// Additional [middleware.Adapter] can be called when a server handles
// a request matching the Route.
type Route struct {
	Path        string
	Method      string
	Handler     http.HandlerFunc
	Middlewares []middleware.Adapter
}

// A Router handles many [Route], directing HTTP requests to the appropriate endpoint.
type Router interface {
	// AuthedRoutes registers the set of Routes as those requiring authentication.
	AuthedRoutes(loginUrl string, logoffUrl string, routes []Route, middlewares ...middleware.Adapter)

	// CatchAll sets up a handler for all routes to funnel to for e.g. maintenace mode.
	CatchAll(handler http.HandlerFunc)

	// Handle applies the [Route] to the Router
	Handle(route Route)

	// HandleNotFound sets the provided [http.HandlerFunc] as the default function
	// for when no other registered Route is matched.
	HandleNotFound(handler http.HandlerFunc)

	// HandleRoutes registers the set of Routes.
	// HandleRoutes calls the provided middlewares before sending a request to the Route.
	HandleRoutes(routes []Route, middlewares ...middleware.Adapter)

	// OnEveryRequest sets the middleware stack to be applied before every request
	//
	// Other methods applying a set of [middleware.Adapter] will always apply theirs
	// after the set defined by OnEveryRequest.
	OnEveryRequest(middlewares ...middleware.Adapter)

	// Subrouter prefixes a Router's handling with the provided string
	Subrouter(prefix string) Router

	SubrouterHost(host string) Router

	// UnauthedRoutes handles the set of Routes
	UnauthedRoutes(routes []Route, middlewares ...middleware.Adapter)

	http.Handler
}

// The DefaultRouter handles HTTP requests to any Routes it is configured with.
//
// DefaultRouter applies the [middleware.ReportPanic] handler to all registered routes.
//
// DefaultRouter routes requests for assets to their location in a standard trails app layout.
// DefaultRouter applies a "Cache-Control" header to responses for assets.
type DefaultRouter struct {
	Env           string
	everyReqStack []middleware.Adapter
	logReq        middleware.Adapter
	*mux.Router
}

// AuthedRoutes registers the set of Routes as those requiring authentication.
// AuthedRoutes applies the given middlewares before performing that check,
// using middleware.RequireAuthed.
//
// middleware.RequireAuthed requires loginUrl and logoffUrl to appropriately
// redirect applicable requests.
// middlweare.RequireAuthed uses key to check whether a user is authenticated or not.
//
// key ought to be the one returned by your keyring.Keyringable.CurrentUserKey.
func (r *DefaultRouter) AuthedRoutes(
	loginUrl,
	logoffUrl string,
	routes []Route,
	middlewares ...middleware.Adapter,
) {
	r.HandleRoutes(routes, append(middlewares, middleware.RequireAuthed(loginUrl, logoffUrl))...)
}

// NewRouter constructs an implementation of [Router] using [DefaultRouter] for the given environment.
//
// TODO(dlk): use provided [fs.FS] and [http.FS] instead of [http.FileServer].
func New(env string, logReq middleware.Adapter) Router {
	r := mux.NewRouter()
	cacheControl := cacheControlMiddleware()

	assetsServer := http.FileServer(http.Dir(assetsPath))

	// NOTE(dlk): direct reqs for the client to its distribution
	r.PathPrefix("/" + assetsPath).Handler(middleware.Chain(
		http.StripPrefix("/"+assetsPath, assetsServer),
		cacheControl,
		logReq,
	))

	return &DefaultRouter{logReq: logReq, Env: env, Router: r}
}

// CatchAll sets up a handler for all routes to funnel to for e.g. maintenace mode.
func (r *DefaultRouter) CatchAll(handler http.HandlerFunc) {
	r.Router.PathPrefix("/").Handler(
		middleware.Chain(
			middleware.ReportPanic(r.Env)(handler),
			r.everyReqStack...,
		),
	)
}

// Handle applies the [Route] to the [Router].
func (r *DefaultRouter) Handle(route Route) {
	r.HandleRoutes([]Route{route})
}

// HandleNotFound sets the provided [http.HandlerFunc] as the default function
// for when no other registered Route is matched.
func (r *DefaultRouter) HandleNotFound(handler http.HandlerFunc) {
	r.Router.NotFoundHandler = middleware.Chain(
		middleware.ReportPanic(r.Env)(handler),
		r.logReq,
	)
}

// HandleRoutes registers the set of Routes on the Router
// and includes all the [middleware.Adapter] on each Route.
// Any [middleware.Adapter] already assigned to a Route is appended to middlewares,
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
// that the [*DefaultRouter] will apply to every request.
func (r *DefaultRouter) OnEveryRequest(middlewares ...middleware.Adapter) {
	r.everyReqStack = append(r.everyReqStack, middlewares...)
}

// ServeHTTP responds to an HTTP request.
func (r *DefaultRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(w, req)
}

func (r *DefaultRouter) SubrouterHost(host string) Router {
	return &DefaultRouter{
		Env:           r.Env,
		Router:        r.Router.Host(host).Subrouter(),
		everyReqStack: r.everyReqStack,
	}
}

// Subrouter constructs a [Router] that handles requests to endpoints matching the prefix.
//
// e.g., r.Subrouter("/api/v1") handles requests to endpoints like /api/v1/users
func (r *DefaultRouter) Subrouter(prefix string) Router {
	return &DefaultRouter{
		Env:           r.Env,
		Router:        r.Router.PathPrefix(prefix).Subrouter(),
		logReq:        r.logReq,
		everyReqStack: r.everyReqStack,
	}
}

// UnauthedRoutes registers the set of Routes as those requiring unauthenticated users.
// It applies the given middlewares before performing that check.
func (r *DefaultRouter) UnauthedRoutes(routes []Route, middlewares ...middleware.Adapter) {
	r.HandleRoutes(routes, append(middlewares, middleware.RequireUnauthed())...)
}

// cacheControlMiddleware helps by adding a "Cache-Control" header to the response.
func cacheControlMiddleware() middleware.Adapter {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
			handler.ServeHTTP(w, r)
		})
	}
}
