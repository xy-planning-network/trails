package ranger

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

// A RangerOption configures a [*Ranger] either (1) directly, immediately upon being called,
// or, (2) in the [OptFollowup] it returns.
// Some RangerOption require data in others and thus an [OptFollowup] can be returned
// in order to be called at a later time when that data is available.
//
// [WithKeyring] is an example of the first.
// An unexported field on the passed in [*Ranger] is updated with the enclosed value.
//
// [WithRouter] is an example of the second.
// An unexported field on the passed in [*Ranger]
// is updated only when the closure it returns is called.
//
// Custom RangerOption can configure exported fields on a [*Ranger]
// to simplify code initializing it.
type RangerOption func(rng *Ranger) (OptFollowup, error)
type OptFollowup func() error

// WithCancelableContext injects ctx and cancel into the trails app.
//
// Neither ctx nor cancel can be nil,
// the [RangerOption] WithCancelableContext returns will return [ErrBadConfig].
func WithCancelableContext(ctx context.Context, cancel context.CancelFunc) RangerOption {
	if ctx == nil || cancel == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			err := fmt.Errorf(
				"%w: WithCancelableContext: neither ctx nor cancel can be nil",
				ErrBadConfig,
			)

			return nil, err
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.ctx = ctx
		rng.cancel = cancel
		setupLog.Debug(fmt.Sprintf("using context %T", ctx), nil)

		return nil, nil
	}
}

// WithDB injects db into the trails app.
//
// WithDB assumes a connection to a database is already been established.
//
// db cannot be nil, the [RangerOption] WithCancelableContext returns will return [ErrBadConfig].
func WithDB(db postgres.DatabaseService) RangerOption {
	if db == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithDB: db cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.db = db
		setupLog.Debug(fmt.Sprintf("using db %T", db), nil)

		return nil, nil
	}
}

// WithEnv injects an [Environment] into the trails app,
// using envVar in one of two ways.
//
// WithEnv first attempts to cast envVar to a valid [Environment],
// for example, "Development".
//
// If this fails, WithEnv uses envVar as a key for reading environment variable,
// for example, "ENVIRONMENT", and then casts the read value into a valid [Environment].
//
// If both fail, WithEnv defaults to injecting [Development].
func WithEnv(envVar string) RangerOption {
	e := Environment(envVar)
	err := e.Valid()
	if err == nil {
		return func(rng *Ranger) (OptFollowup, error) {
			rng.env = e
			setupLog.Debug(fmt.Sprintf("using env %s", e), nil)

			return nil, nil

		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.env = envVarOrEnv(envVar, Development)
		setupLog.Debug(fmt.Sprintf("using env %s", rng.env), nil)

		return nil, nil
	}
}

// WithKeyring injects k into the trails app.
//
// k cannot be nil, the [RangerOption] WithKeyring returns will return [ErrBadConfig].
func WithKeyring(k keyring.Keyringable) RangerOption {
	if k == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithKeyring: k cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.kr = k
		setupLog.Debug(fmt.Sprintf("using keyring %T", k), nil)

		return nil, nil
	}
}

// WithLogger injects l into the trails app.
//
// l cannot be nil, the [RangerOption] WithLogger returns will return [ErrBadConfig].
func WithLogger(l logger.Logger) RangerOption {
	if l == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithLogger: l cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.Logger = l
		setupLog = l
		setupLog.Debug(fmt.Sprintf("using logger %T", l), nil)

		return nil, nil
	}
}

// WithResponder injects r into the trails app.
// The [RangerOption] WithResponder returns constructs an [OptFollowup],
// which finally performs the injection,
// in order to allow the dependencies for [*resp.Responder] to themselves be injected.
//
// r cannot be nil, the [RangerOption] WithResponder returns will return [ErrBadConfig].
func WithResponder(r *resp.Responder) RangerOption {
	if r == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithResponder: r cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			rng.Responder = r
			setupLog.Debug("using responder", nil)

			return nil
		}, nil
	}
}

// WithRouter injects r into the trails app.
// The [RangerOption] WithResponder returns constructs an [OptFollowup],
// which finally performs the injection,
// in order to allow the dependencies for [router.Router] to themselves be injected.
//
// r cannot be nil, the [RangerOption] WithRouter returns will return [ErrBadConfig].
func WithRouter(r router.Router) RangerOption {
	if r == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithRouter: r cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			if rng.srv == nil {
				rng.srv = defaultServer(rng.ctx)
			}

			rng.Router = r
			rng.srv.Handler = r

			setupLog.Debug(fmt.Sprintf("using router %T", r), nil)
			setupLog.Debug(fmt.Sprintf("using server %T", rng.srv), nil)

			return nil
		}, nil
	}
}

// WithServer injects s into the trails app.
//
// s cannot be nil, the [RangerOption] WithServer returns will return [ErrBadConfig].
func WithServer(s *http.Server) RangerOption {
	if s == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithServer: s cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		old := rng.srv
		rng.srv = s

		if old != nil {
			rng.srv.Handler = old.Handler
		}

		return nil, nil
	}
}

// WithSessionStore injects store into the trails app.
//
// store cannot be nil, the [RangerOption] WithSessionStore returns will return [ErrBadConfig].
func WithSessionStore(store session.SessionStorer) RangerOption {
	if store == nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: WithSessionStore: store cannot be nil", ErrBadConfig)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.sessions = store
		setupLog.Debug(fmt.Sprintf("using session store %T", store), nil)

		return nil, nil
	}
}

// WithUserSessions injects users into the trails app.
//
// When WithUserSessions is called, it overrides the default [middleware.UserStorer].
// The default [middleware.UserStorer] gets or creates a [postgres.DatabaseService] connection.
//
// users cannot be nil, the [RangerOption] WithUserSessions returns will return [ErrBadConfig].
func WithUserSessions(users middleware.UserStorer) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.users = users

		setupLog.Debug(fmt.Sprintf("using user store %T", users), nil)

		return nil, nil
	}
}

// ShutdownFn stops a service that a trails app should also gracefully shutdown
// when the trails web server is itself shutting down.
//
// Calling [*Ranger.Shutdown] calls any ShutdownFn injected into the app.
type ShutdownFn func(context.Context) error

// WithShutdowns injects shutdownFns into the trails app.
func WithShutdowns(shutdownFns ...ShutdownFn) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.shutdowns = shutdownFns

		return nil, nil
	}
}
