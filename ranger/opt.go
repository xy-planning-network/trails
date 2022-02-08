package ranger

import (
	"context"
	"net/http"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

// A RangerOption configures a *Ranger either (1) directly, immediately upon being called
// or (2) in the optFollowup it returns.
// Some RangerOptions require data in others and thus an optFollowup can be returned
// in order to be called at a later time when that data is available.
//
// WithKeyring is an example of the first.
// An unexported field on the passed in *Ranger is updated with the enclosed value.
//
// WithRouter is an example of the second.
// An unexported field on the passed in *Ranger
// is updated only when the closure it returns is called.
type RangerOption func(rng *Ranger) (OptFollowup, error)
type OptFollowup func() error

// WithContext exposes the provided context.Context to the trails app.
func WithContext(ctx context.Context) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.ctx = ctx
		return nil, nil
	}
}

// WithDB exposes the provided postgres.DatabaseService to the trails app.
//
// WithDB assumes a connection has already been established.
func WithDB(db postgres.DatabaseService) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.DB = db
		return nil, nil
	}
}

// WithEnv casts the provided string into a valid Environment,
// or, reads from the ENVIRONMENT environment variable a valid Environment.
// WithEnv then exposes that Environment in the the Ranger.Env field.
//
// If both fail, the default Environment is set to Development.
func WithEnv(envVar string) RangerOption {
	e := Environment(envVar)
	err := e.Valid()
	if err == nil {
		return func(rng *Ranger) (OptFollowup, error) {
			rng.Env = e
			return nil, nil

		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		rng.Env = envVarOrEnv(envVar, Development)
		return nil, nil
	}
}

// WithKeyring exposes the provided keyring.Keyringable to the trails app.
func WithKeyring(k keyring.Keyringable) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.Keyring = k
		return nil, nil
	}
}

// WithLogger exposes the provided logger.Logger to the trails app.
func WithLogger(l logger.Logger) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.Logger = l
		return nil, nil
	}
}

// WithResponder constructs a followup option that, when called,
// exposes the *resp.Responder to the trails app.
func WithResponder(r *resp.Responder) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			rng.Responder = r
			return nil
		}, nil
	}
}

// WithRouter constructs a followup option that, when called,
// exposes the router.Router to the trails app.
func WithRouter(r router.Router) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			// TODO(dlk): best approach? need to track 2 fields?
			if rng.srv == nil {
				rng.srv = defaultServer()
			}
			rng.Router = r
			rng.srv.Handler = r

			return nil
		}, nil
	}
}

// WithSessionStore exposes the session.SessionStorer to the trails app.
func WithSessionStore(store session.SessionStorer) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		rng.SessionStorer = store
		return nil, nil
	}
}

// WithServer exposes the *http.Server to the trails app.
func WithServer(s *http.Server) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		old := rng.srv
		rng.srv = s

		if old != nil {
			rng.srv.Handler = old.Handler
		}

		return nil, nil
	}
}
