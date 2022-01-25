package ranger

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/xy-planning-network/trails/domain"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
	"gorm.io/gorm"
)

const (
	// Base URL defaults
	baseURLEnvVar  = "BASE_URL"
	defaultBaseURL = "http://localhost"

	// Environment defaults
	envFile   = ".env"
	envEnvVar = "ENVIRONMENT"

	// Log defaults
	logLevelEnvVar = "LOG_LEVEL"
	defaultLogLvl  = logger.LogLevelInfo

	// Database defaults
	dbHostEnvVar  = "DATABASE_HOST"
	defaultDBHost = "localhost"
	dbNameEnvVar  = "DATABASE_NAME"
	dbPassEnvVar  = "DATABSE_PASSWORD"
	dbPortEnvVar  = "DATABASE_PORT"
	defaultDBPort = "5432"
	dbURLEnvVar   = "DATABASE_URL"
	dbUserEnvVar  = "DATABASE_USER"

	dbTestHostEnvVar  = "DATABASE_TEST_HOST"
	defaultDBTestHost = "localhost"
	dbTestNameEnvVar  = "DATABASE_TEST_NAME"
	dbTestPassEnvVar  = "DATABASE_TEST_PASSWORD"
	dbTestPortEnvVar  = "DATABASE_TEST_PORT"
	defaultDBTestPort = "5432"
	dbTestURLEnvVar   = "DATABASE_TEST_URL"
	dbTestUserEnvVar  = "DATABASE_TEST_USER"

	// Default template files
	// TODO(dlk): embed a default template dir
	defaultLayoutDir    = "tmpl/layout"
	defaultAuthedTmpl   = defaultLayoutDir + "/authenticated_base.tmpl"
	defaultUnauthedTmpl = defaultLayoutDir + "/unauthenticated_base.tmpl"

	// Web server defaults
	serverReadTimeoutEnvVar   = "SERVER_READ_TIMEOUT"
	DefaultServerReadTimeout  = 5 * time.Second
	serverIdleTimeoutEnvVar   = "SERVER_IDLE_TIMEOUT"
	DefaultServerIdleTimeout  = 120 * time.Second
	serverWriteTimeoutEnvVar  = "SERVER_WRITE_TIMEOUT"
	DefaultServerWriteTimeout = 5 * time.Second

	// Session defaults
	sessionAuthKeyEnvVar    = "SESSION_AUTH_KEY"
	sessionEncryptKeyEnvVar = "SESSION_ENCRYPTION_KEY"

	// Default context keys
	defaultSessionCtxKey     = keyring.Key("trails-ranger-default-session-key")
	defaultCurrentUserCtxKey = keyring.Key("trails-ranger-default-current-user-key")
	defaultRequestIDCtxKey   = keyring.Key("trails-ranger-default-request-id-key")
)

// defaultOpts returns the default RangerOptions used in every call to NewRanger.
//
// These can be overwrittern using With* functional options
// or building off Default* configurations.
func defaultOpts() []RangerOption {
	return []RangerOption{
		DefaultContext(),
		DefaultLogger(),
		WithEnv(envEnvVar),
		DefaultKeyring(),
		DefaultSessionStore(),
		DefaultRouter(),
	}
}

// DefaultContext constructs a RangerOption initiates a new, base context.Context
// for the trails app.
func DefaultContext() RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		return WithContext(context.Background())(rng)
	}
}

// DefaultDB constructs a RangerOption that connects to a database
// using default configuration environment variables
// and runs the list of postgres.Migrations passed in.
func DefaultDB(list []postgres.Migration) RangerOption {
	var cfg *postgres.CxnConfig
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			switch rng.Env {
			case Testing: // NOTE(dlk): this is an unexpected case since go test does not reach this
				cfg = &postgres.CxnConfig{
					Host:     envVarOrString(dbTestHostEnvVar, defaultDBTestHost),
					IsTestDB: true,
					Name:     os.Getenv(dbTestNameEnvVar),
					Password: os.Getenv(dbTestPassEnvVar),
					Port:     envVarOrString(dbTestPortEnvVar, defaultDBTestPort),
					User:     os.Getenv(dbTestUserEnvVar),
				}

			default:
				if url := os.Getenv(dbURLEnvVar); url != "" {
					cfg = &postgres.CxnConfig{IsTestDB: false, URL: url}
				} else {
					cfg = &postgres.CxnConfig{
						Host:     envVarOrString(dbHostEnvVar, defaultDBHost),
						IsTestDB: false,
						Name:     os.Getenv(dbNameEnvVar),
						Password: os.Getenv(dbPassEnvVar),
						Port:     envVarOrString(dbPortEnvVar, defaultDBPort),
						User:     os.Getenv(dbUserEnvVar),
					}
				}
			}

			db, err := postgres.Connect(cfg, list)
			if err != nil {
				return err
			}

			fn, err := WithDB(postgres.NewService(db))(rng)
			if err != nil {
				return err
			}

			return fn()
		}, nil
	}
}

// DefaultKeyring constructs a RangerOption that applies the default context keys
// and those keys passed in to the Ranger.
func DefaultKeyring(keys ...keyring.Keyable) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		kr := keyring.NewKeyring(
			defaultSessionCtxKey,
			defaultCurrentUserCtxKey,
			append(keys, defaultRequestIDCtxKey)...,
		)

		return WithKeyring(kr)(rng)
	}
}

// DefaultLogger constructs a RangerOption that applies the default logger (logger.DefaultLogger)
// to the Ranger.
func DefaultLogger(opts ...logger.LoggerOptFn) RangerOption {
	logLvl := envVarOrLogLevel(logLevelEnvVar, logger.LogLevelInfo)
	args := []logger.LoggerOptFn{
		logger.WithLevel(logLvl),
	}
	for _, opt := range opts {
		args = append(args, opt)
	}

	return func(rng *Ranger) (OptFollowup, error) {
		return WithLogger(logger.NewLogger(args...))(rng)
	}
}

// DefaultParser constructs a RangerOption that configures a default HTML template parser to be used
// when responding to HTTP requests with *resp.Responder.Html.
//
// BASE_URL ought to be set when the default http://localhost:3000 is not wanted.
//
// DefaultParser makes available these functions in an HTML template:
//
// - "env"
// - "nonce"
// - "rootUrl"
// - "packTag"
// - "isDevelopment"
// - "isStaging"
// - "isProduction"
func DefaultParser(files fs.FS, opts ...template.ParserOptFn) RangerOption {
	u, err := url.ParseRequestURI(envVarOrString(baseURLEnvVar, defaultBaseURL))
	if err != nil {
		return func(_ *Ranger) (OptFollowup, error) {
			return nil, fmt.Errorf("%w: failed parsing %s: %s", ErrBadConfig, baseURLEnvVar, err)
		}
	}

	return func(rng *Ranger) (OptFollowup, error) {
		args := []template.ParserOptFn{
			template.WithFn(template.Env(rng.Env.String())),
			template.WithFn(template.Nonce()),
			template.WithFn(template.RootUrl(u)),
			template.WithFn("packTag", template.TagPackerModern(rng.Env.String(), files)),
			template.WithFn("isDevelopment", func() bool { return rng.Env == Development }),
			template.WithFn("isStaging", func() bool { return rng.Env == Staging }),
			template.WithFn("isProduction", func() bool { return rng.Env == Production }),
		}

		for _, opt := range opts {
			args = append(args, opt)
		}

		rng.p = template.NewParser(files, args...)

		return nil, nil
	}

}

// DefaultResponder constructs a RangerOption that returns a followup option
// configuring the *Ranger.Responder to be used by http.Handlers.
//
// BASE_URL ought to be set when the default http://localhost:3000 is not wanted.
//
// DefaultResponder delays directly configuring the *Ranger under construction
// in order to avail itself of data - such as template.Parser -
// that other RangerOptions configure.
func DefaultResponder(opts ...resp.ResponderOptFn) RangerOption {
	u, err := url.ParseRequestURI(envVarOrString(baseURLEnvVar, defaultBaseURL))
	if err != nil {
		err = fmt.Errorf("%w: failed parsing %s: %s", ErrBadConfig, baseURLEnvVar, err)
		return func(rng *Ranger) (OptFollowup, error) { return nil, err }
	}

	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			if rng.p == nil {
				if _, err := DefaultParser(os.DirFS("."))(rng); err != nil {
					return err
				}
			}

			args := []resp.ResponderOptFn{
				resp.WithRootUrl(u.String()),
				resp.WithLogger(rng.Logger),
				resp.WithParser(rng.p),
				resp.WithAuthTemplate(defaultAuthedTmpl),
				resp.WithUnauthTemplate(defaultUnauthedTmpl),
				resp.WithSessionKey(rng.Keyring.SessionKey()),
				resp.WithUserSessionKey(rng.Keyring.CurrentUserKey()),
			}
			for _, opt := range opts {
				args = append(args, opt)
			}

			r := resp.NewResponder(args...)

			fn, err := WithResponder(r)(rng)
			if err != nil {
				return err
			}

			return fn()
		}, nil
	}
}

type defaultUserStorer struct {
	postgres.DatabaseService
}

// GetByID retrieves the middleware.User matching the ID.
func (store defaultUserStorer) GetByID(id uint) (middleware.User, error) {
	user := new(domain.User)
	if err := store.FindByID(user, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = fmt.Errorf("%w: User %d", ErrNotExist, id)
		}

		return nil, err
	}

	return user, nil
}

// DefaultRouter constructs a RangerOption that returns a followup option
// configuring the *Ranger.Router to be used by the web server.
func DefaultRouter() RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		var fn func() error
		var err error
		fn, err = DefaultResponder()(rng)
		if err != nil {
			return nil, err
		}

		r := router.NewRouter(rng.Env.String())
		r.OnEveryRequest(
			middleware.RateLimit(middleware.NewVisitors()),
			middleware.ForceHTTPS(rng.Env.String()),
			middleware.RequestID(rng.Keyring.Key(defaultRequestIDCtxKey.Key())),
			middleware.InjectIPAddress(),
			middleware.LogRequest(rng),
			middleware.InjectSession(rng.sess, rng.Keyring.SessionKey()),
			middleware.CurrentUser(
				rng.Responder,
				defaultUserStorer{rng.DB},
				rng.Keyring.SessionKey(),
				rng.Keyring.CurrentUserKey(),
			),
		)

		return func() error {
			if err := fn(); err != nil {
				return err
			}

			fn, err = WithRouter(r)(rng)
			if err != nil {
				return err
			}

			return fn()
		}, nil
	}
}

// DefaultSessionStore constructs a RangerOption that configures the SessionStore
// to be used for storing session data.
//
// DefaultSessionStore relies on two env vars:
// - "SESSION_AUTH_KEY"
// - "SESSION_ENCRYPTION_KEY"
//
// These must be valid hex encoded values.
//
// If these values do not exist, DefaultSessionStore will generate new, random keys
// and save those to the env var file for later use.
func DefaultSessionStore(opts ...session.ServiceOpt) RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		args := []session.ServiceOpt{
			session.WithCookie(),
			session.WithMaxAge(3600 * 24 * 7),
		}

		for _, opt := range opts {
			args = append(args, opt)
		}

		auth := os.Getenv(sessionAuthKeyEnvVar)
		if auth == "" {
			k := fmt.Sprintf("%x", securecookie.GenerateRandomKey(64))
			if err := writeToEnvFile(sessionAuthKeyEnvVar, k); err != nil {
				return nil, err
			}

			auth = k
		}

		encrypt := os.Getenv(sessionEncryptKeyEnvVar)
		if encrypt == "" {
			k := fmt.Sprintf("%x", securecookie.GenerateRandomKey(32))
			if err := writeToEnvFile(sessionEncryptKeyEnvVar, k); err != nil {
				return nil, err
			}

			encrypt = k
		}

		store, err := session.NewStoreService(rng.Env.String(), auth, encrypt, args...)
		if err != nil {
			return nil, err
		}

		return WithSessionStore(store)(rng)
	}
}

// defaultServer constructs a default *http.Server.
func defaultServer() *http.Server {
	return &http.Server{
		Addr:         ":3000", // TODO(dlk): configurable
		ReadTimeout:  envVarOrDuration(serverReadTimeoutEnvVar, DefaultServerReadTimeout),
		IdleTimeout:  envVarOrDuration(serverIdleTimeoutEnvVar, DefaultServerIdleTimeout),
		WriteTimeout: envVarOrDuration(serverWriteTimeoutEnvVar, DefaultServerWriteTimeout),
	}
}

// writeToEnvFile appends the key value pair to the default env var file, .env
func writeToEnvFile(key, value string) error {
	file, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// TODO(dlk): smart error handling,  e.g., os.ErrPermission?
		return err
	}

	line := fmt.Sprintf("%s=%s\n", key, value)
	if _, err := file.WriteString(line); err != nil {
		return err
	}

	return nil

}
