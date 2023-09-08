package ranger

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/xy-planning-network/tint"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// Base URL defaults
	BaseURLEnvVar = "BASE_URL"

	// App metadata
	AppDescEnvVar    = "APP_DESCRIPTION"
	AppTitleEnvVar   = "APP_TITLE"
	ContactUsEnvVar  = "CONTACT_US_EMAIL"
	defaultContactUs = "hello@xyplanningnetwork.com"

	// Environment defaults
	environmentEnvVar = "ENVIRONMENT"

	// Log defaults
	logLevelEnvVar  = "LOG_LEVEL"
	defaultLogLvl   = slog.LevelInfo
	logJSONEnvVar   = "LOG_JSON"
	defaultLogJSON  = false
	sentryDsnEnvVar = "SENTRY_DSN"

	// Database defaults
	dbHostEnvVar         = "DATABASE_HOST"
	defaultDBHost        = "localhost"
	dbNameEnvVar         = "DATABASE_NAME"
	dbPassEnvVar         = "DATABASE_PASSWORD"
	dbPortEnvVar         = "DATABASE_PORT"
	defaultDBPort        = "5432"
	dbSSLModeEnvVar      = "DATABASE_SSLMODE"
	defaultDBSSLMode     = "prefer"
	dbURLEnvVar          = "DATABASE_URL"
	dbUserEnvVar         = "DATABASE_USER"
	dbMaxIdleCxnsEnvVar  = "DATABASE_MAX_IDLE_CXNS"
	defaultDBMaxIdleCxns = 1

	// Default HTML template files
	defaultTmplDir               = "tmpl"
	defaultErrTmpl               = defaultTmplDir + "/error.tmpl"
	defaultLayoutDir             = defaultTmplDir + "/layout"
	defaultAdditionalScriptsTmpl = defaultLayoutDir + "/additional_scripts.tmpl"
	defaultAuthedTmpl            = defaultLayoutDir + "/authenticated_base.tmpl"
	defaultUnauthedTmpl          = defaultLayoutDir + "/unauthenticated_base.tmpl"
	defaultVueTmpl               = defaultLayoutDir + "/vue.tmpl"
	defaultVueScriptsTmpl        = defaultLayoutDir + "/vue_scripts.tmpl"

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
	SessionAuthKeyEnvVar    = "SESSION_AUTH_KEY"
	SessionEncryptKeyEnvVar = "SESSION_ENCRYPTION_KEY"

	// Test defaults
	dbTestHostEnvVar     = "DATABASE_TEST_HOST"
	defaultDBTestHost    = "localhost"
	dbTestNameEnvVar     = "DATABASE_TEST_NAME"
	dbTestPassEnvVar     = "DATABASE_TEST_PASSWORD"
	dbTestPortEnvVar     = "DATABASE_TEST_PORT"
	defaultDBTestPort    = "5432"
	dbTestURLEnvVar      = "DATABASE_TEST_URL"
	dbTestUserEnvVar     = "DATABASE_TEST_USER"
	dbTestSSLModeEnvVar  = "DATABASE_TEST_SSLMODE"
	defaultDBTestSSLMode = "prefer"
)

var (
	defaultBaseURL = "http://" + DefaultHost + DefaultPort

	//go:embed tmpl/*
	tmpls embed.FS
)

// NewPostgresConfig constructs a *postgres.CxnConfig appropriate to the given environment.
// Confer the DATABASE env vars for usage.
func NewPostgresConfig(env trails.Environment) *postgres.CxnConfig {
	var cfg *postgres.CxnConfig
	url := os.Getenv(dbURLEnvVar)
	switch {
	case env.IsTesting():
		cfg = &postgres.CxnConfig{
			Host:     trails.EnvVarOrString(dbTestHostEnvVar, defaultDBTestHost),
			IsTestDB: true,
			Name:     os.Getenv(dbTestNameEnvVar),
			Password: os.Getenv(dbTestPassEnvVar),
			Port:     trails.EnvVarOrString(dbTestPortEnvVar, defaultDBTestPort),
			SSLMode:  trails.EnvVarOrString(dbTestSSLModeEnvVar, defaultDBTestSSLMode),
			User:     os.Getenv(dbTestUserEnvVar),
		}

	case url == "":
		cfg = &postgres.CxnConfig{
			Host:     trails.EnvVarOrString(dbHostEnvVar, defaultDBHost),
			IsTestDB: false,
			Name:     os.Getenv(dbNameEnvVar),
			Password: os.Getenv(dbPassEnvVar),
			Port:     trails.EnvVarOrString(dbPortEnvVar, defaultDBPort),
			SSLMode:  trails.EnvVarOrString(dbSSLModeEnvVar, defaultDBSSLMode),
			User:     os.Getenv(dbUserEnvVar),
		}

	default:
		cfg = &postgres.CxnConfig{IsTestDB: false, URL: url}
	}

	cfg.MaxIdleCxns = trails.EnvVarOrInt(dbMaxIdleCxnsEnvVar, defaultDBMaxIdleCxns)

	return cfg
}

// defaultDB connects to a Postgres database
// using default configuration environment variables
// and runs the list of [postgres.Migration] passed in.
func defaultDB(env trails.Environment, list []postgres.Migration) (postgres.DatabaseService, error) {
	db, err := postgres.Connect(NewPostgresConfig(env), list, env)
	if err != nil {
		return nil, err
	}

	return postgres.NewService(db), nil
}

// defaultAppLogger constructs a [tlog.Logger] configured for use in the application.
func defaultAppLogger(env trails.Environment, output io.Writer) logger.Logger {
	slogger := newSlogger(trails.AppLogKind, env, output)
	l := logger.New(slogger)
	l.Debug("setting up app logger", nil)
	if dsn := os.Getenv(sentryDsnEnvVar); dsn != "" {
		l = logger.NewSentryLogger(env, l, dsn)
		l.Debug("using SentryLogger for app logger", nil)
	}

	slog.SetDefault(slogger)

	return l
}

// defaultHTTPLogger constructs a [*log/slog.Logger] for use in HTTP router logging.
func defaultHTTPLogger(env trails.Environment, output io.Writer) *slog.Logger {
	sl := newSlogger(trails.HTTPLogKind, env, output)
	sl.Debug("setting up HTTP router logger")

	return sl
}

// defaultWorkerLogger constructs a [*log/slog.Logger] for use in Faktory worker logging.
func defaultWorkerLogger(env trails.Environment) logger.Logger {
	slogger := newSlogger(trails.WorkerLogKind, env, os.Stdout)
	l := logger.New(slogger)
	l.Debug("setting up worker logger", nil)
	if dsn := os.Getenv(sentryDsnEnvVar); dsn != "" {
		l = logger.NewSentryLogger(env, l, dsn)
		l.Debug("using SentryLogger for worker logger", nil)
	}

	return l
}

// newSlogger toggles contructing the specific [*log/slog.Logger]
// from the given parameters.
func newSlogger(kind slog.Value, env trails.Environment, out io.Writer) *slog.Logger {
	lvl := new(slog.LevelVar)
	lvl.Set(trails.EnvVarOrLogLevel(logLevelEnvVar, slog.LevelInfo))

	useJSON := !env.IsDevelopment() || trails.EnvVarOrBool(logJSONEnvVar, defaultLogJSON)
	kindStr := kind.String()
	isApp := kindStr == trails.AppLogKind.String()
	isHTTP := kindStr == trails.HTTPLogKind.String()
	isWorker := kindStr == trails.WorkerLogKind.String()

	var handler slog.Handler
	switch {
	case useJSON && (isApp || isWorker):
		opts := &slog.HandlerOptions{
			AddSource:   true,
			Level:       lvl,
			ReplaceAttr: logger.TruncSourceAttr,
		}

		handler = slog.NewJSONHandler(out, opts)

	case !useJSON && (isApp || isWorker):
		opts := &tint.Options{
			AddSource:  true,
			Level:      lvl,
			TimeFormat: "2006-01-02 15:04:05.000",
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				a = logger.ColorizeLevel(groups, a)
				return logger.TruncSourceAttr(groups, a)
			},
		}
		handler = tint.NewHandler(out, opts)

	case isHTTP && useJSON:
		opts := &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				a = logger.DeleteLevelAttr(groups, a)
				return logger.DeleteMessageAttr(groups, a)
			},
		}
		handler = slog.NewJSONHandler(out, opts)

	case isHTTP && !useJSON:
		opts := &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				a = logger.DeleteLevelAttr(groups, a)
				return logger.DeleteMessageAttr(groups, a)
			},
		}
		handler = slog.NewTextHandler(out, opts)

	}

	handler = handler.WithAttrs([]slog.Attr{
		{Key: trails.LogKindKey, Value: kind},
	})

	return slog.New(handler)
}

// defaultParser constructs a *template.Parser to be used
// when responding to HTTP requests with [*http/resp.Responder.Html].
//
// defaultParser makes available these functions in an HTML template:
//
//   - "env"
//   - "metadata"
//   - "description" returns the value set by the APP_DESCRIPTION env var
//   - "title" returns the value set by the APP_TITLE env var
//   - "nonce"
//   - "rootUrl"
//   - "packTag"
//   - "isDevelopment"
//   - "isStaging"
//   - "isProduction"
func defaultParser(env trails.Environment, url *url.URL, files fs.FS, m Metadata) *template.Parser {
	p := template.NewParser([]fs.FS{files, tmpls})
	p = p.AddFn(template.Env(env))
	p = p.AddFn("isDevelopment", env.IsDevelopment)
	p = p.AddFn("isStaging", env.IsStaging)
	p = p.AddFn("isProduction", env.IsProduction)
	p = p.AddFn(m.templateFunc())
	p = p.AddFn(template.Nonce())
	p = p.AddFn("packTag", template.TagPacker(env, os.DirFS(".")))
	p = p.AddFn(template.RootUrl(url))

	return p
}

// defaultResponder configures the [*resp.Responder] to be used by http.Handlers.
func defaultResponder(l logger.Logger, url *url.URL, p *template.Parser, contact string) *resp.Responder {
	args := []resp.ResponderOptFn{
		resp.WithAdditionalScriptsTemplate(defaultAdditionalScriptsTmpl),
		resp.WithAuthTemplate(defaultAuthedTmpl),
		resp.WithContactErrMsg(fmt.Sprintf(session.ContactUsErr, contact)),
		resp.WithErrTemplate(defaultErrTmpl),
		resp.WithLogger(l),
		resp.WithParser(p),
		resp.WithRootUrl(url.String()),
		resp.WithUnauthTemplate(defaultUnauthedTmpl),
		resp.WithVueTemplate(defaultVueTmpl),
		resp.WithVueScriptsTemplate(defaultVueScriptsTmpl),
	}

	return resp.NewResponder(args...)
}

// defaultRouter constructs a [router.Router] to be used by the web server.
func defaultRouter(
	env trails.Environment,
	baseURL *url.URL,
	responder *resp.Responder,
	mws []middleware.Adapter,
) router.Router {
	route := router.NewRouter(env.String())
	route.OnEveryRequest(mws...)
	route.HandleNotFound(http.HandlerFunc(func(wx http.ResponseWriter, rx *http.Request) {
		if strings.Contains(rx.Header.Get("Accept"), "text/html") && rx.URL.Path != baseURL.Path {
			responder.Redirect(wx, rx, resp.ToRoot())
			return
		}

		wx.WriteHeader(http.StatusNotFound)
	}))

	return route
}

// defaultSessionStore constructs a SessionStorer to be used for storing session data.
//
// defaultSessionStore relies on three env vars:
//   - APP_TITLE
//   - SESSION_AUTH_KEY
//   - SESSION_ENCRYPTION_KEY
//
// Both KEY env vars be valid hex encoded values; cf. [encoding/hex].
func defaultSessionStore(env trails.Environment, appName string) (session.SessionStorer, error) {
	appName = cases.Lower(language.English).String(appName)
	appName = regexp.MustCompile(`[,':]`).ReplaceAllString(appName, "")
	appName = regexp.MustCompile(`\s`).ReplaceAllString(appName, "-")

	cfg := session.Config{
		AuthKey:     os.Getenv(SessionAuthKeyEnvVar),
		EncryptKey:  os.Getenv(SessionEncryptKeyEnvVar),
		Env:         env,
		SessionName: "trails-" + appName,
	}

	args := []session.ServiceOpt{
		session.WithCookie(),
		session.WithMaxAge(3600 * 24 * 7),
	}

	return session.NewStoreService(cfg, args...)
}

// defaultServer constructs a default [*http.Server].
func defaultServer(ctx context.Context) *http.Server {
	port := trails.EnvVarOrString(portEnvVar, DefaultPort)
	if port[0] != ':' {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:         port,
		IdleTimeout:  trails.EnvVarOrDuration(serverIdleTimeoutEnvVar, DefaultServerIdleTimeout),
		ReadTimeout:  trails.EnvVarOrDuration(serverReadTimeoutEnvVar, DefaultServerReadTimeout),
		WriteTimeout: trails.EnvVarOrDuration(serverWriteTimeoutEnvVar, DefaultServerWriteTimeout),
	}
	if ctx != nil {
		srv.BaseContext = func(_ net.Listener) context.Context { return ctx }
	}

	return srv
}
