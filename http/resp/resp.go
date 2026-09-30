package resp

import (
	"log/slog"
	"net/http"
	"reflect"

	"github.com/go-json-experiment/jsonsplit"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

func init() {
	jsonsplit.GlobalCodec.AutoDetectOptions = true
	jsonsplit.GlobalCodec.ReportDifference = func(d jsonsplit.Difference) {
		t := d.GoType
		if tt, ok := d.GoValue.(jsonSchema); ok {
			if ttt, ok := tt.D.(postgres.PagedData); ok {
				t = reflect.TypeOf(ttt.Items)
				if t.Kind() == reflect.Slice {
					t = t.Elem()
				}
			} else {
				t = reflect.TypeOf(tt.D)
			}
		}

		var opts []string
		for name := range d.OptionNames() {
			opts = append(opts, name)
		}

		// NOTE(dlk): We're living with a some init and global var funkiness
		// as slog & jsonsplit intersect here.
		// As long as we can grep the logs and parse a structured format,
		// we'll be alright.
		slog.Warn(
			"json-diff",
			slog.String("caller", d.Caller),
			slog.String("func", d.Func),
			slog.Any("options", opts),
			slog.String("type", t.PkgPath()+"."+t.Name()),
		)
	}
	jsonsplit.GlobalCodec.SetMarshalCallMode(jsonsplit.CallBothButReturnV1)
}

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
