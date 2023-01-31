/*
Package resp provides a high-level API for responding to HTTP requests.

The core of the package revolves around the interplay between a Responder,
which ought to be configured for broad contexts (i.e., application-wide),
and a Response, which lives at the http.Handler level.

Within a handler, a Responder constructs a Response and finally executes a response in one of these ways:
- rendering HTML templates
- rendering JSON data
- redirecting
- writing error messages/codes

Responders and Responses draw upon configuration happening through functional options.
To avoid needless error throwing, both may silently ignore options that are irrelevant
or would put them in invalid states.
In some cases, a Responder may emit logs warning of incorrect usage,
enabling a developer to remediate these mistakes within their workflow.

However, some incorrect use cannot be fixed and all of the forms of response made available
by a Responder (e.g., Html, Json, Redirect) can return meaningful errors.

The Responder is responsible for providing any data a Response may need to do its work correctly.
Notably, a Responder contains data on default templates, keys used for an *http.Request.Context,
the web app's root URL and so forth.
ResponderOptFns carry out configuring a Responder and are unlikely to be used within a handler.
Instead, it is expected these feature in a web app's router setup steps.
*/
package resp
