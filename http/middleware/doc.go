/*
The middleware package defines what a middleware is in trails and a set of basic middlewares.

The available middlewares are:
- CORS
- CurrentUser
- ForceHTTPS
- InjectSession
- LogRequest
- RateLimit
- RequestID

Due to the amount of configuration required, middleware does not provide a default middleware chain
Instead, the following can be copy-pasted:

	vs := middleware.NewVisitors()
	adpts := []middleware.Adapter{
		middleware.RateLimit(vs),
		middleware.ForceHTTPS(env),
		middleware.RequestID(requestIDKey),
		middleware.LogRequest(log),
		middleware.InjectSession(sessionStore, sessionKey),
		middleware.CurrentUser(responder, userStore, userKey),
	}

*/
package middleware
