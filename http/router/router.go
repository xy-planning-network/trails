package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xy-planning-network/trails/http/middleware"
)

const (
	assetsPath       = "/assets/"
	assetsPublicPath = "client/public/"
	clientDistPath   = "client/dist/"
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

// Router routes requests for resources to their location in a standard trails app layout.
type Router struct {
	Env           string
	everyReqStack []middleware.Adapter
	logReq        middleware.Adapter
	r             *mux.Router
}

// New constructs a [*Router] for the given environment.
//
// TODO(dlk): use provided [fs.FS] and [http.FS] instead of [http.FileServer].
func New(env string, logReq middleware.Adapter) *Router {
	r := mux.NewRouter()
	cacheControl := cacheControlMiddleware()

	assetsServer := http.FileServer(http.Dir(assetsPublicPath))
	clientServer := http.FileServer(http.Dir(clientDistPath))

	// NOTE(dlk): direct reqs for the client to its distribution
	r.PathPrefix("/" + clientDistPath).Handler(middleware.Chain(
		http.StripPrefix("/"+clientDistPath, clientServer),
		cacheControl,
		logReq,
	))

	// NOTE(dlk): direct reqs for assets to public path
	r.PathPrefix(assetsPath).Handler(middleware.Chain(
		http.StripPrefix(assetsPath, assetsServer),
		cacheControl,
		logReq,
	))

	return &Router{logReq: logReq, Env: env, r: r}
}

// AuthedRoutes registers the set of Routes as those requiring authentication.
// AuthedRoutes applies the given middlewares before performing that check,
// using middleware.RequireAuthed.
//
// middleware.RequireAuthed requires loginUrl and logoffUrl to appropriately
// redirect applicable requests.
func (r *Router) AuthedRoutes(
	loginUrl,
	logoffUrl string,
	routes []Route,
	middlewares ...middleware.Adapter,
) {
	mws := append(middlewares, middleware.RequireAuthed(loginUrl, logoffUrl))
	r.HandleRoutes(routes, mws...)
}

// CatchAll sets up a handler for all routes to funnel to for e.g. maintenance mode.
func (r *Router) CatchAll(handler http.HandlerFunc) {
	r.r.PathPrefix("/").Handler(
		middleware.Chain(
			middleware.ReportPanic(r.Env)(handler),
			r.everyReqStack...,
		),
	)
}

// Handle applies the [Route] to the [*Router].
func (r *Router) Handle(route Route) {
	r.HandleRoutes([]Route{route})
}

// HandleNotFound sets the provided [http.HandlerFunc] as the default function
// for when no other registered Route is matched.
func (r *Router) HandleNotFound(handler http.HandlerFunc) {
	r.r.NotFoundHandler = middleware.Chain(
		middleware.ReportPanic(r.Env)(handler),
		r.logReq,
	)
}

// HandleRoutes registers the set of Routes on the Router
// and includes all the [middleware.Adapter] on each Route.
// Any [middleware.Adapter] already assigned to a Route is appended to middlewares,
// so are called after the default set.
func (r *Router) HandleRoutes(routes []Route, middlewares ...middleware.Adapter) {
	for _, route := range routes {
		mws := append(r.everyReqStack, middlewares...)
		mws = append(mws, route.Middlewares...)
		handler := middleware.Chain(middleware.ReportPanic(r.Env)(route.Handler), mws...)
		r.r.Handle(route.Path, handler).Methods(route.Method)
	}

}

// OnEveryRequest appends the middlewares to the existing stack
// that the [*Router] will apply to every request.
func (r *Router) OnEveryRequest(middlewares ...middleware.Adapter) {
	r.everyReqStack = append(r.everyReqStack, middlewares...)
}

// ServeHTTP responds to an HTTP request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

func (r *Router) SubrouterHost(host string) *Router {
	return &Router{
		Env:           r.Env,
		r:             r.r.Host(host).Subrouter(),
		everyReqStack: r.everyReqStack,
	}
}

// Subrouter constructs a [Router] that handles requests to endpoints matching the prefix.
//
// e.g., r.Subrouter("/api/v1") handles requests to endpoints like /api/v1/users
func (r *Router) Subrouter(prefix string) *Router {
	return &Router{
		Env:           r.Env,
		r:             r.r.PathPrefix(prefix).Subrouter(),
		logReq:        r.logReq,
		everyReqStack: r.everyReqStack,
	}
}

// UnauthedRoutes registers the set of Routes as those requiring unauthenticated users.
// It applies the given middlewares before performing that check.
func (r *Router) UnauthedRoutes(routes []Route, middlewares ...middleware.Adapter) {
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
