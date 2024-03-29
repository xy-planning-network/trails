package resp

import (
	"net/http"

	"github.com/xy-planning-network/trails/logger"
)

// newLogContext helps structure a logger.LogContext from the provided parts.
func newLogContext(r *http.Request, err error, data any, user logger.LogUser) *logger.LogContext {
	if r == nil && err == nil && data == nil && user == nil {
		return nil
	}

	ctx := new(logger.LogContext)
	if r != nil {
		ctx.Request = r
	}

	if err != nil {
		ctx.Error = err
	}

	if mapped, ok := data.(map[string]any); ok {
		ctx.Data = mapped
	}

	if user != nil {
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

	return CurrentUser(u)(d, r)
}

// stubLogger enables setting up a Responder without logging,
// specifically for unit tests.
type stubLogger struct{}

func (l stubLogger) AddSkip(i int) logger.Logger          { return l }
func (l stubLogger) Skip() int                            { return 0 }
func (l stubLogger) Debug(_ string, _ *logger.LogContext) { return }
func (l stubLogger) Info(_ string, _ *logger.LogContext)  { return }
func (l stubLogger) Warn(_ string, _ *logger.LogContext)  { return }
func (l stubLogger) Error(_ string, _ *logger.LogContext) { return }
