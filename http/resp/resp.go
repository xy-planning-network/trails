package resp

import (
	"net/http"

	"github.com/xy-planning-network/trails/logger"
)

// getLogContext helps structure a logger.LogContext from the provided parts.
func getLogContext(r *http.Request, err error, data, u interface{}) *logger.LogContext {
	if r == nil && err == nil && data == nil && u == nil {
		return nil
	}

	ctx := new(logger.LogContext)
	if r != nil {
		ctx.Request = r
	}
	if err != nil {
		ctx.Error = err
	}
	if mapped, ok := data.(map[string]interface{}); ok {
		ctx.Data = mapped
	}
	if user, ok := u.(logger.LogUser); ok {
		ctx.User = user
	}
	return ctx
}

// populateUser helps pull a user up out of the *Response.r.Context
// and into the *Response itself.
func populateUser(d Responder, r *Response) error {
	if r.user != nil {
		return nil
	}

	u, err := d.CurrentUser(r.r.Context())
	if err != nil || u == nil {
		return ErrNoUser
	}

	return User(u)(d, r)
}
