# Go on the Trails

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/xy-planning-network/trails.svg)](https://pkg.go.dev/github.com/xy-planning-network/trails)

## What's Trails
Trails unifies the patterns and solutions [XY Planning Network](https://xyplanningnetwork.com) developed to power a handful of web applications. We at XYPN prefer the slower method of walking the trails and staying closer to the dirt over something speedier on the road. Nevertheless, Trails has opinions and removes boilerplate when it can. Trails will be in v0 for the foreseeable future.

Trails provides libraries for quickly building web applications that have standard, well-defined web application needs, such as managing user sessions or routing based on user authorization. It defines the concepts needed for solving those problems through interfaces and provides default implementations of those so development can begin immediately.

## So, what's in here?
### `ranger/`
A trails app is set managed and guided by a `\*ranger.Ranger`. A `\*ranger.Ranger` composes the different tools trails makes available and provides opinionated defaults. It is as simple as:
```go 
package main

import (
        "github.com/xy-planning-network/trails/resp"
        "github.com/xy-planning-network/trails/ranger"
)

type handler struct {
        *resp.Responder
}

func (h *handler) GetHelloWorld(w http.ResponseWriter, r *http.Request) {
        h.Raw(w, r, Data("Hello, World!"))
}

func main() {
        rng := ranger.New()

        h := &handler{rng.Responder}

        rng.Handle(router.Route{Method: http.MethodGet, Path: "/", Handler: h})
}
```

### `http/`
It may be trails' Ranger is too opinionated for your use case. Very well, trails pushes each and every element of a web app into its own module. These can be used on their own as a toolkit, rather than a framework.

In `http/` we find trails' web server powered by a router, middleware stack, HTML template rendering, user session management and a high-level declarative API for crafting HTTP responses. Let's get this setup!

#### `http/router`
The first thing Trails does is initialize an HTTP router:
```golang
package main

import (
        "net/http"

        "github.com/xy-planning-network/trails/http/router"
)

func main() {
        r := router.NewRouter("DEVELOPMENT")
        r.Handle(router.Path{Path: "/", Method: http.MethodGet, Handler: getRoot}) // this and other functions in other examples would be defined elsewhere 😅
        http.ListenAndServe(":3000", r)
}
```
Not too useful, yet, just one route at `/` to direct requests to. But, this shows Trails' router implements `http.Handler`! We don't want to stray too far away from the standard library.

Let's get a few more routes in there.

Trails' router encourages registering routes in logically similar groups.
```golang
package main

import (
        "net/http"

        "github.com/xy-planning-network/trails/http/router"
)

func main() {
        r := router.NewRouter("DEVELOPMENT")
        base := []router.Route{
                {Path: "/login", Method: http.MethodGet, Handler: getLogin},
                {Path: "/logoff", Method: http.MethodGet, Handler: getLogoff},
                {Path: "/password/reset", Method: http.MethodGet, Handler: getPasswordReset},
                {Path: "/password/reset", Method: http.MethodPost, Handler: resetPassword},
        }
        r.HandleRoutes(base)

        http.ListenAndServe(":3000", r)
}
```

🎉 We did it! Our Trails app serves up 4 distinct routes. 🎉

It is often the case that many routes for a web server share identical middleware stacks, which aid in directing, redirecting, or adding contextual information to a request. It is also often the case that small errors can lead to registering a route incorrectly, thereby unintentionally exposing a resource or not collecting data necessary for actually handling a request.

The example above does not utilize any middleware, which can be quickly rectified by using Trails' `middleware` library:

```golang
package main

import (
        "github.com/xy-planning-network/trails/http/middleware"
        "github.com/xy-planning-network/trails/http/router"
)

func main() {
        r := router.NewRouter("DEVELOPMENT")
        r.OnEveryRequest(middleware.InjectIPAddress())

        policies := []router.Route{
                {Path: "/terms", Method: http.MethodGet, Handler: getTerms},
                {Path: "/privacy-policy", Method: http.MethodGet, Handler: getPrivacyPolicy},
        }
        r.HandleRoutes(policies)

        base := []router.Route{
                {Path: "/login", Method: http.MethodGet, Handler: getLogin},
                {Path: "/logoff", Method: http.MethodGet, Handler: getLogoff},
                {Path: "/password/reset", Method: http.MethodGet, Handler: getPasswordReset},
                {Path: "/password/reset", Method: http.MethodPost, Handler: resetPassword},
        }
        r.HandleRoutes(
                base,
                middleware.LogRequest(logger.DefaultLogger()),
        )
}
```

We've added middlewares in two places in two different ways.

First, we use `Router.OnEveryRequest` to set a middleware that grabs the originating request's IP address on every single `Route`. Next, we include a middleware that logs the request when we also register or `base` routes. This logger will run only when a request matches one of those `base` routes.

Let's start getting fancy 🍸.

In our Trails app, we don't want our users who've already logged in to access neither the login page or password reset page - they should only be able to reset their password from a settings page. Furthermore, only authenticated users should be able to access the logoff endpoint. We can use Trails baked-in support for authentication to reorganize our routing:

```golang
package main

import (
        "github.com/xy-planning-network/trails/http/middleware"
        "github.com/xy-planning-network/trails/http/router"
)

func main() {
        env := "DEVELOPMENT"

        sessionstore := session.NewStoreService(env, "ABCD", "ABCD") // Read more about me in http/session

        r := router.NewRouter(env)
        r.OnEveryRequest(
                middleware.InjectIPAddress(),
                middleware.InjectSession(sessionstore, 🗝), // 🗝: read more about managing keys used for a *http.Request.Context in http/ctx
        )

        policies := []router.Route{
                {Path: "/terms", Method: http.MethodGet, Handler: getTerms},
                {Path: "/privacy-policy", Method: http.MethodGet, Handler: getPrivacyPolicy},
        }
        r.HandleRoutes(policies)

        unauthed := []router.Route{
                {Path: "/login", Method: http.MethodGet, Handler: getLogin},
                {Path: "/password/reset", Method: http.MethodGet, Handler: getPasswordReset},
                {Path: "/password/reset", Method: http.MethodPost, Handler: resetPassword},
        }
        r.UnauthedRoutes(🗝, unauthed)

        authed := []router.Route{
                {Path: "/logoff", Method: http.MethodGet, Handler: getLogoff},
                {Path: "/settings", Method: http.MethodGet, Handler: getSettings},
                {Path: "/settings", Method: http.MethodPut, Handler: updateSettings},
        }
        r.AuthedRoutes(🗝, "/login", "/logoff", authed)
}
```

Organizing routes around middleware stacks, especially those relating to authentication and authorization, can aid in eliminating subtle bugs.

#### `http/resp`
Given the `Router` directed a request correctly, Trails provides a high-level API for crafting responses in an HTTP handler. An HTTP handler uses a `Responder` to join together application-wide configuration and handler-specific needs. This standardizes responses across the web app enabling clients to rely on the HTTP headers, status codes, data schemas, etc. coming from Trails. We initialize a `Responder` using functional options and make that available to all our handlers:

```golang
package main

import (
	"embed"
	"net/http"

        "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/template"
)

//go:embed *.tmpl
var files embed.FS

type handler struct {
        *resp.Responder
}

func (h *handler) getLogin(w http.ResponseWriter, r *http.Request) {
        h.Html(w, r, resp.Tmpl("root.tmpl"))
}

func main() {
        p := template.NewParser(files) // Read more about me in http/template
        d := resp.NewResponder(resp.WithParser(p))
        h := &handler{d}
        r := router.NewRouter("DEVELOPMENT")
        r.Handle(router.Route{Path: "/", Method: http.MethodGet, Handler: r.getLogin})
}
```

Let's elide over the use of `embed` and `trails/http/template` for now in order to focus on this line in our handler:

```golang
h.Html(w, r, resp.Tmpl("root.tmpl"))
```

With the \*resp.Responder embedded in our `handler`, we can utilize it's `Html` method to render HTML templates and respond with that data. Using a `resp.Fn`, we set the template to render. If that template needs some additional values, we can provide those with `resp.Data`:

```golang
func (h *handler) getLogin(w http.ResponseWriter, r *http.Request) {
        hello := map[string]any{"welcomeMsg": "Hello, World!"}
        err := h.Html(w, r, resp.Tmpl("root.tmpl"), resp.Data(hello))
        if err != nil {
                h.Err(w, r, err)
                return
        }
}
```

These `resp.Fn` functional options are highly flexible. Some are generic - such as `resp.Code`. Some compose together multiple options - such as `resp.GenericErr`. Even more, some are specialized - such as `resp.Props` - for apps leveraging the full suite of features available in Trails.

Notably, a `Responder` concludes the lifecycle of an HTTP request by writing a response in one of these ways:
  - Err
    - `*Responder.Err` provides a backstop for malformed calls to `*Responder` methods by wrapping std lib's `http.Error`.
  - Html
    - `*Responder.Html` renders templates written in Go's `html/template` syntax.
  - Json
    - `*Responder.Json` renders data in JSON format.
  - Redirect
    - `*Responder.Redirect` redirects a request to another endpoint, a wrapper around `http.Redirect`. 

## Trees
Trails integrates with XYPN's open-source Vue component library, [Trees](http://github.com/xy-planning-network/trees), in two quick steps. Setup is as simple as defining the path to your base Vue template, passing in the path to your base Vue template using `resp.WithVueTemplate`, and include that template with `resp.Vue` when using `resp.(*Responder).Html`.

## What needs to be done?
- [x] Database connection
- [x] Database migrations
- [x] Routing
- [x] Middlewares
- [x] Response handling
- [x] Session management
- [ ] Form scaffolding
- [ ] Vue 3 integrations
- [x] Logging
- [ ] Authentication/Authorization
- [ ] Parsing + sending emails

## HELP 🔥🔥🔥
- My web server just keeps send `200`s and nothing else!
  - All examples have been tested (minus bugs!) and so use the convenience of not checking the `error` a `*http.Responder` method may return. When in doubt, start handling those errors. Instead of:
    ```go
      func myHandler(w http.ResponseWriter, r *http.Request) {
        Html(w, r, Tmpls("my-root.tmpl"))
      }
    ```
    try
    ```go
      func myHandler(w http.ResponseWriter, r *http.Request) {
        if err := Html(w, r, Tmpls("my-root.tmpl")); err != nil {
          Err(w, r, err)
        }
      }
    ```

## Pioneers
Below are "pioneers" who make our work easier and deserve more credit than just an import in the `go.mod`:

- [Gorilla Web Toolkit](https://github.com/gorilla)
  - To implement a web server, Trails relies on the Gorilla Web Toolkit.

### Links
- [XY Planning Network](https://www.xyplanningnetwork.com)
- [Trees](https://github.com/xy-planning-network/trees)
