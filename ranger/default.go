package ranger

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

const (
	// Base URL defaults
	baseURLEnvVar = "BASE_URL"

	// App metadata
	appDescEnvVar   = "APP_DESCRIPTION"
	defaultAppDesc  = "A trails app"
	appTitleEnvVar  = "APP_TITLE"
	defaultAppTitle = "trails"

	// Environment defaults
	environmentEnvVar = "ENVIRONMENT"

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
	defaultTmplDir      = "tmpl"
	defaultErrTmpl      = defaultTmplDir + "/error.tmpl"
	defaultLayoutDir    = defaultTmplDir + "/layout"
	defaultAuthedTmpl   = defaultLayoutDir + "/authenticated_base.tmpl"
	defaultUnauthedTmpl = defaultLayoutDir + "/unauthenticated_base.tmpl"
	defaultVueTmpl      = defaultLayoutDir + "/vue.tmpl"

	// Web server defaults
	DefaultHost               = "localhost"
	hostEnvVar                = "HOST"
	DefaultPort               = ":3000"
	portEnvVar                = "PORT"
	serverReadTimeoutEnvVar   = "SERVER_READ_TIMEOUT"
	DefaultServerReadTimeout  = 5 * time.Second
	serverIdleTimeoutEnvVar   = "SERVER_IDLE_TIMEOUT"
	DefaultServerIdleTimeout  = 120 * time.Second
	serverWriteTimeoutEnvVar  = "SERVER_WRITE_TIMEOUT"
	DefaultServerWriteTimeout = 5 * time.Second

	// Session defaults
	sessionAuthKeyEnvVar    = "SESSION_AUTH_KEY"
	sessionEncryptKeyEnvVar = "SESSION_ENCRYPTION_KEY"
	useUserSessionsEnvVar   = "USE_USER_SESSIONS"

	// Default context keys
	defaultSessionCtxKey     = keyring.Key("trails-ranger-default-session-key")
	defaultCurrentUserCtxKey = keyring.Key("trails-ranger-default-current-user-key")
	defaultRequestIDCtxKey   = keyring.Key("trails-ranger-default-request-id-key")
)

var (
	defaultBaseURL, _ = url.ParseRequestURI("http://" + DefaultHost + DefaultPort)
	setupLog          logger.Logger

	//go:embed tmpl/*
	tmpls embed.FS
)

// defaultOpts returns the default RangerOptions used in every call to NewRanger.
//
// These can be overwrittern using With* functional options
// or building off Default* configurations.
func defaultOpts() []RangerOption {
	return []RangerOption{
		DefaultLogger(),
		DefaultContext(),
		DefaultKeyring(),
		DefaultRouter(),
	}
}

// DefaultContext constructs a RangerOption initiates a new, base context.Context
// for the trails app.
func DefaultContext() RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		ctx, cancel := context.WithCancel(context.Background())
		return WithCancelableContext(ctx, cancel)(rng)
	}
}

// DefaultDB constructs a RangerOption that connects to a database
// using default configuration environment variables
// and runs the list of postgres.Migrations passed in.
func DefaultDB(list []postgres.Migration) RangerOption {
	var cfg *postgres.CxnConfig
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			switch rng.env {
			case trails.Testing: // NOTE(dlk): this is an unexpected case since go test does not reach this
				cfg = &postgres.CxnConfig{
					Host:     trails.EnvVarOrString(dbTestHostEnvVar, defaultDBTestHost),
					IsTestDB: true,
					Name:     os.Getenv(dbTestNameEnvVar),
					Password: os.Getenv(dbTestPassEnvVar),
					Port:     trails.EnvVarOrString(dbTestPortEnvVar, defaultDBTestPort),
					User:     os.Getenv(dbTestUserEnvVar),
				}

			default:
				if url := os.Getenv(dbURLEnvVar); url != "" {
					cfg = &postgres.CxnConfig{IsTestDB: false, URL: url}
				} else {
					cfg = &postgres.CxnConfig{
						Host:     trails.EnvVarOrString(dbHostEnvVar, defaultDBHost),
						IsTestDB: false,
						Name:     os.Getenv(dbNameEnvVar),
						Password: os.Getenv(dbPassEnvVar),
						Port:     trails.EnvVarOrString(dbPortEnvVar, defaultDBPort),
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
	logLvl := trails.EnvVarOrLogLevel(logLevelEnvVar, logger.LogLevelInfo)
	args := []logger.LoggerOptFn{
		logger.WithLevel(logLvl),
	}
	args = append(args, opts...)

	return func(rng *Ranger) (OptFollowup, error) {
		return WithLogger(logger.New(args...))(rng)
	}
}

// DefaultParser constructs a RangerOption that configures a default HTML template parser to be used
// when responding to HTTP requests with *resp.Responder.Html.
//
// files can be nil, resulting in the parser using the local directory to find templates.
//
// BASE_URL ought to be set when the default http://localhost:3000 is not wanted.
//
// DefaultParser makes available these functions in an HTML template:
//
// - "env"
// - "metadata"
// - "nonce"
// - "rootUrl"
// - "packTag"
// - "isDevelopment"
// - "isStaging"
// - "isProduction"
func DefaultParser(files fs.FS, opts ...template.ParserOptFn) RangerOption {
	if files == nil {
		files = os.DirFS(".")
	}

	return func(rng *Ranger) (OptFollowup, error) {
		if rng.url == nil {
			rng.url = trails.EnvVarOrURL(baseURLEnvVar, defaultBaseURL)
		}

		title := trails.EnvVarOrString(appTitleEnvVar, defaultAppTitle)
		desc := trails.EnvVarOrString(appDescEnvVar, defaultAppDesc)

		args := []template.ParserOptFn{
			template.WithFn(template.Env(rng.env)),
			template.WithFn(template.Nonce()),
			template.WithFn(template.RootUrl(rng.url)),
			template.WithFn("isDevelopment", func() bool { return rng.env.IsDevelopment() }),
			template.WithFn("isStaging", func() bool { return rng.env.IsStaging() }),
			template.WithFn("isProduction", func() bool { return rng.env.IsProduction() }),
			template.WithFn("metadata", func(key string) string {
				return map[string]string{
					"description": desc,
					"title":       title,
				}[key]
			}),
			template.WithFn("packTag", template.TagPacker(rng.env, files)),
		}

		args = append(args, opts...)

		rng.p = template.NewParser([]fs.FS{files, tmpls}, args...)

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
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			if rng.url == nil {
				rng.url = trails.EnvVarOrURL(baseURLEnvVar, defaultBaseURL)
			}

			if rng.p == nil {
				if _, err := DefaultParser(os.DirFS("."))(rng); err != nil {
					return err
				}
			}

			args := []resp.ResponderOptFn{
				resp.WithRootUrl(rng.url.String()),
				resp.WithLogger(rng.Logger),
				resp.WithParser(rng.p),
				resp.WithAuthTemplate(defaultAuthedTmpl),
				resp.WithErrTemplate(defaultErrTmpl),
				resp.WithUnauthTemplate(defaultUnauthedTmpl),
				resp.WithVueTemplate(defaultVueTmpl),
			}
			if rng.kr != nil {
				args = append(
					args,
					resp.WithSessionKey(rng.kr.SessionKey()),
					resp.WithUserSessionKey(rng.kr.CurrentUserKey()),
				)
			}

			args = append(args, opts...)

			r := resp.NewResponder(args...)

			fn, err := WithResponder(r)(rng)
			if err != nil {
				return err
			}

			return fn()
		}, nil
	}
}

// DefaultRouter constructs a RangerOption that returns a followup option
// configuring the *Ranger.Router to be used by the web server.
func DefaultRouter() RangerOption {
	return func(rng *Ranger) (OptFollowup, error) {
		return func() error {
			fn, err := DefaultResponder()(rng)
			if err != nil {
				return err
			}

			if err := fn(); err != nil {
				return err
			}

			mws := make([]middleware.Adapter, 0)
			if rng.env.IsProduction() {
				mws = append(
					mws,
					middleware.ForceHTTPS(rng.env.String()),
				)
			}

			mws = append(
				mws,
				middleware.RequestID(defaultRequestIDCtxKey),
				middleware.InjectIPAddress(),
				middleware.LogRequest(rng.Logger),
			)

			if rng.sessions == nil {
				if _, err = DefaultSessionStore()(rng); err != nil {
					return err
				}
			}

			mws = append(
				mws,
				middleware.InjectSession(rng.sessions, rng.kr.SessionKey()),
				middleware.CurrentUser(
					rng.Responder,
					rng.userstore,
					rng.kr.SessionKey(),
					rng.kr.CurrentUserKey(),
				),
			)

			r := router.NewRouter(rng.env.String())
			r.OnEveryRequest(mws...)
			r.HandleNotFound(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
				if strings.Contains(rx.Header.Get("Accept"), "text/html") && rx.URL.Path != rng.url.Path {
					rng.Redirect(wx, rx, resp.ToRoot())
					return
				}

				wx.WriteHeader(http.StatusNotFound)
			}))

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

		args = append(args, opts...)

		ll := setupLog
		if sl, ok := ll.(logger.SkipLogger); ok {
			ll = sl.AddSkip(sl.Skip() + 2)
		}

		auth := os.Getenv(sessionAuthKeyEnvVar)
		if auth == "" {
			ll.Warn("missing required value for "+sessionAuthKeyEnvVar, nil)
			return nil, nil
		}

		encrypt := os.Getenv(sessionEncryptKeyEnvVar)
		if encrypt == "" {
			ll.Warn("missing required value for "+sessionEncryptKeyEnvVar, nil)
			return nil, nil
		}

		store, err := session.NewStoreService(
			rng.env,
			auth,
			encrypt,
			string(defaultSessionCtxKey),
			string(defaultCurrentUserCtxKey),
			args...,
		)
		if err != nil {
			return nil, err
		}

		return WithSessionStore(store)(rng)
	}
}

// defaultServer constructs a default *http.Server.
func defaultServer(ctx context.Context) *http.Server {
	port := trails.EnvVarOrString(portEnvVar, DefaultPort)
	if port[0] != ':' {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:         port,
		ReadTimeout:  trails.EnvVarOrDuration(serverReadTimeoutEnvVar, DefaultServerReadTimeout),
		IdleTimeout:  trails.EnvVarOrDuration(serverIdleTimeoutEnvVar, DefaultServerIdleTimeout),
		WriteTimeout: trails.EnvVarOrDuration(serverWriteTimeoutEnvVar, DefaultServerWriteTimeout),
	}
	if ctx != nil {
		srv.BaseContext = func(_ net.Listener) context.Context { return ctx }
	}

	return srv
}
