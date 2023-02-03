/*

Package router defines what an HTTP server is and a default implementation of it.

The package defines what a web server router does in Trails through [Router]
and a default implementation of it [*DefaultRouter].
[*DefaultRouter] utilizes [mux.Mux] for its implementation,
and so functions as thin wrapper around that pacakge.

A [Router] leverages a standardized data model - a [Route] -
when registering how requests should be routed.
A path and an HTTP method comprise a [Route].
An implementation of [http.Handler] is the function called when a request matches a Route.
Before a request gets to a handler, though,
any middlewares added to the Route are called in the order they appear.

It is often the case that many routes for a web server share identical middleware stacks,
which aid in directing, redirecting, or adding contextual information to a request.
It is also often the case that small errors can lead to registering a route incorrectly,
thereby unintentionally exposing a resource or not collecting data necessary for actually handling a request.
Thus, a [Router] provides conveniences for making a single call to register many logically associated Routes.

A Router expects two such groups of routes:
those pointing to resources, alternatively, outside of or behind authentication barriers.
The UnauthedRoutes and AuthedRoutes methods ensure routes are registered in the appropriate way, consequently.

*/
package router
