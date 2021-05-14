This draft proposes an API for composing HTTP responses through functional options.

## Overview
### Current State
HTTP responses are highly flexible and context-dependent. However, the std libs `http` pkg is too "low-level". We end up with repeated code, duplicated boilerplate, and unintentional bugs from dev errors.

In `second-child` & `college-try`, convenience methods standardizing common workflows for HTTP responses were created to solve that first problem. There are only a handful, so generally easy to memorize. They are:
1. JSON responses:
   - `respondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{})`:
       - responds with JSON-encoded data and the passed in status code
1. Render responses that employ `buildVueResponse` to structure the passed in data for `vue.tmpl`:
    - `respondUnauthenticated(w http.ResponseWriter, r *http.Request, templateToRender string, data interface{})`:
        - renders the passed in template using the passed in data by first wrapping it in the `unauthenticated_base.tmpl`
    - `respondAuthenticated(w http.ResponseWriter, r *http.Request, templateToRender string, data interface{})`:
        - renders the passed in template used the passed in data by first wrapping it in the `authenticated_base.tmpl`
1. Redirect responses:
    - `redirectWithCustomError(w http.ResponseWriter, r *http.Request, err error, data map[string]interface{}, url, flashType, flashMessage string)`:
        - redirects to the specified url after logging the error and setting the specified flash in the session.
    - `redirectWithGenericError(w http.ResponseWriter, r *http.Request, err error, data map[string]interface{}, url string)`:
        - calls `redirectWithCustomError` with `flashType` error.
    - `redirectOnSuccess(w http.ResponseWriter, r *http.Request, url, flashMessage string)`:
        - redirects to the specified url after setting a success flash on the session.

###  Problem
There are a few instances where we call `http.Redirect` directly, but, otherwise, these methods solved all HTTP response needs in the previous two projects. The multi-purpose nature of these functions, nevertheless, leads to an ambiguous API interface. Consider: `nil` is a reasonible `error` to pass to `redirectWithGenericError`. What happens when one does so? To answer that question, one must review the actual function body or documentation (if available).

As well, consider our lock-in into using the `unauthenticated_base` and `authenticated_base` templates in those projects. Thus far, there is no known use case for not needing these, but the fetters exist. The template render workflow is not trivial and needing to duplicate it simply to support an alternative render path would be a chore rife with potential error.

### Solution
A `Responder` provides an application-level configuration for how to respond to HTTP requests. With each HTTP response, a `Responder` applies fields configured at an application-level, requisite parts of the `*http.Request` being responded to, and all functional options to the `http.ResponseWriter` provided by the handler. Let's jump straight into an example showing both short and long forms. In the below example, before redirecting, an error is logged and a flash error is set in the user's session:
```go
// note . import to simplify below example, open question whether this should be the pattern to replicate
import . "github.com/xy-planning-network/trails/resp"

type Handler struct {
            // other fields
            Responder
}

func (h *Handler) myHandler(w http.ResponseWriter, r *http.Request) {
        // some work that declares the diverse identifiers passed into the functional options

        // short-form
        Redirect(w, r, Url(GetLoginURL), GenericErr(err), Redirect())

        // long-form
        Redirect(
                w, r,
                User(cu),
                Err(err),
                Url(GetLoginURL),
                Code(http.StatusInternalServerError),
                Flash(FlashError, ErrorMessage),
        )
}
```
The first, short-form uses the convenience method `GenericErr`. This way, we need not manually set the user, error, status code, and flash on the response since that functional option does it all for us. The long-form spells out what those options are to get a better look at what we can do with this library.

Under the hood, `Redirect` initializes and builds up a `*Response` object that stores data passed in from functional options and then, finally, sends a redirect response. For example, `Url(u string)` parses the URL in `u` and assigns to `*Response.url` to ensure a valid URL can be redirected to. 

Options that require calling a previous functional option will throw an `error` if this requirement has not been met. Each `Responder` method leverages a loop to attempt to heal these situations by calling those options throwing errors (in the same order they were passed to the method) again until all issues are resolved _or_ a set of options are left over that require remediation by the caller. If it is kicked off, a warning is logged, allowing a developer to repair the situation before it lands in a production environment.

Notably, all of these functional options can validly be called outside `Responder.Do`. Accordingly, error statuses can be inspected by the caller and handled on an option-by-option basis, if so desired. This would be a more advanced usage that ought to be kept in mind while developing this package.

#### Current proposed functional options:
```go
Authed()
Unauthed()
Code(c int)
Data(d map[string]interface{})
Err(e error)
Flash(class, msg string)
GenericErr(e error)
Params(key, val string)
Props(p map[string]interface{})
Success(msg string)
Tmpls(ts ...string)
User(u domain.User)
Url(u string)
Warn(msg string)
Vue(entry string)
RespondErr(e error)
Json()
Redirect()
Render()
```

### Trade-offs
Different order of functional options do not necessarily create the same result rendering it difficult to know what to expect. This is especially true for options that themselves compose other options (e.g., `GenericErr`). It is unclear the extent to which unit tests will be able to catch potential bugs, but reinforcing proper unit testing may eliminate this trade-off.

This new approach contends with a philosophy of having "one way to do things". Calls for an HTTP response could end up looking different (say, different order of functional options) even if producing the exact same result.

### Next Steps
1. Templates
1. Unit tests.
1. In it's current draft, it is possible to code out an invalid `Respond` by forgetting to include a terminal method that writes to the `http.ResponseWriter`. This can be solved by having the appropriate methods stored on the underlying `*Response` object and calling it after processing all the other options, if it exists, erroring if it does not. An alternative approach is defined below.

### Potential changes
#### Remove needless execution
Compare these two `respond` calls:
```go
Do(w, r, Err(e), Props(p), Unauthed(), Tmpls(someParsedTmpl), Render())
Do(w, r, err(e), Props(p), Unauthed, Tmpls(someParsedTmpl), Render)
```
The second instance omits actually calling functional options that need no initialization. The question here, then, is whether the mixture of, alternatively, executed and referenced functions leads to a confusing API surface. While the compiler and IDE would obviate a developer composing a `Do` incorrectly, it may simply look like magic why some options are executed while others are not. Referring to the function body or documentation to clarify that difference is an extra, unnecessary step in the development experience.

#### Terminal response as methods on `Responder` instead of standalone functions
As mentioned in the trade-offs, one must include the final function that actually writes to the `http.ResponseWriter` for that action to occur. Those functions could be proper methods, however.

Compare:
```go
// Note on-the-fly initialization of Responder for clarity's sake; normally would be initialized already
(Responder{}).Do(w, r, Err(e), Props(p), Unauthed(), Tmpls(someParsedTmpl), Render())
(Responder{}).Render(w, r, Err(e), Props(p), Unauthed(), Tmpls(someParsedTmpl))
```
Instead of a generic `Do` function that merely controls building the `*Response` and hopes that it's a valid object, `Render`, in the above example, would perform those duties and then actually render the templates. At the moment, this would mean three response functions: `Render`, `Redirect` & `Json` replacing `Do`.

### `http/api`
By and large all OK responses in `http/api` would go from this:
```go
h.respondJSON(w, r, http.StatusOK, data)
```
to:
```go
h.Do(w, r, Data(data), Json())
```
and all not OK statuses require logging would move from:
```go
h.Logger.Error(err.Error(), logCtx)
h.respondJSON(w, r, http.StatusInternalServerError)
```
to:
```go
h.Do(w, r, Err(err), Json())
```
Swap out `Err(err)` for `Code(http.StatusInternalServerError` if no error needs logging. 

Make explicit a couple of wins, here.

- Instead of needing to log and then respond, the err method will take care of your logging
- Don't have two methods rendering templates anymore, which posed a risk of diverging over time
- I don't have to keep autocompleting nil, nil when writing a redirectWithGenericError response
