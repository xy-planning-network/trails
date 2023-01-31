package ranger

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

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
	baseURLEnvVar = "BASE_URL"

	// App metadata
	appDescEnvVar    = "APP_DESCRIPTION"
	defaultAppDesc   = "A trails app"
	appTitleEnvVar   = "APP_TITLE"
	defaultAppTitle  = "trails"
	contactUsEnvVar  = "CONTACT_US_EMAIL"
	defaultContactUs = "hello@xyplanningnetwork.com"

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

	// Default HTML template files
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
	sessionNameEnvVar       = "SESSION_NAME"
	useUserSessionsEnvVar   = "USE_USER_SESSIONS"

	// Test defaults
	dbTestHostEnvVar  = "DATABASE_TEST_HOST"
	defaultDBTestHost = "localhost"
	dbTestNameEnvVar  = "DATABASE_TEST_NAME"
	dbTestPassEnvVar  = "DATABASE_TEST_PASSWORD"
	dbTestPortEnvVar  = "DATABASE_TEST_PORT"
	defaultDBTestPort = "5432"
	dbTestURLEnvVar   = "DATABASE_TEST_URL"
	dbTestUserEnvVar  = "DATABASE_TEST_USER"
)

var (
	defaultBaseURL, _ = url.ParseRequestURI("http://" + DefaultHost + DefaultPort)
	setupLog          logger.Logger

	//go:embed tmpl/*
	tmpls embed.FS
)

// defaultDB connects to a Postgres database
// using default configuration environment variables
// and runs the list of postgres.Migrations passed in.
func defaultDB(env trails.Environment, list []postgres.Migration) (postgres.DatabaseService, error) {
	var cfg *postgres.CxnConfig
	url := os.Getenv(dbURLEnvVar)
	switch {
	case env.IsTesting(): // NOTE(dlk): this is an unexpected case since go test does not reach this
		cfg = &postgres.CxnConfig{
			Host:     trails.EnvVarOrString(dbTestHostEnvVar, defaultDBTestHost),
			IsTestDB: true,
			Name:     os.Getenv(dbTestNameEnvVar),
			Password: os.Getenv(dbTestPassEnvVar),
			Port:     trails.EnvVarOrString(dbTestPortEnvVar, defaultDBTestPort),
			User:     os.Getenv(dbTestUserEnvVar),
		}

	case url == "":
		cfg = &postgres.CxnConfig{
			Host:     trails.EnvVarOrString(dbHostEnvVar, defaultDBHost),
			IsTestDB: false,
			Name:     os.Getenv(dbNameEnvVar),
			Password: os.Getenv(dbPassEnvVar),
			Port:     trails.EnvVarOrString(dbPortEnvVar, defaultDBPort),
			User:     os.Getenv(dbUserEnvVar),
		}

	default:
		cfg = &postgres.CxnConfig{IsTestDB: false, URL: url}
	}

	db, err := postgres.Connect(cfg, list)
	if err != nil {
		return nil, err
	}

	return postgres.NewService(db), nil
}

// defaultLogger constructs a logger.Logger.
func defaultLogger() logger.Logger {
	logLvl := trails.EnvVarOrLogLevel(logLevelEnvVar, defaultLogLvl)
	args := []logger.LoggerOptFn{
		logger.WithLevel(logLvl),
	}

	return logger.New(args...)
}

// defaultParser constructs a template.Parser to be used
// when responding to HTTP requests with *resp.Responder.Html.
//
// defaultParser makes available these functions in an HTML template:
//
// - "env"
// - "metadata"
// - "nonce"
// - "rootUrl"
// - "packTag"
// - "isDevelopment"
// - "isStaging"
// - "isProduction"
func defaultParser(env trails.Environment, url *url.URL, files fs.FS) template.Parser {
	title := trails.EnvVarOrString(appTitleEnvVar, defaultAppTitle)
	desc := trails.EnvVarOrString(appDescEnvVar, defaultAppDesc)

	args := []template.ParserOptFn{
		template.WithFn(template.Env(env)),
		template.WithFn(template.Nonce()),
		template.WithFn(template.RootUrl(url)),
		template.WithFn("isDevelopment", func() bool { return env.IsDevelopment() }),
		template.WithFn("isStaging", func() bool { return env.IsStaging() }),
		template.WithFn("isProduction", func() bool { return env.IsProduction() }),
		template.WithFn("metadata", func(key string) string {
			return map[string]string{
				"description": desc,
				"title":       title,
			}[key]
		}),
		template.WithFn("packTag", template.TagPacker(env, files)),
	}

	return template.NewParser([]fs.FS{files, tmpls}, args...)
}

// defaultResponder configures the*&resp.Responder to be used by http.Handlers.
func defaultResponder(l logger.Logger, url *url.URL, p template.Parser, ctxKeys []trails.Key) *resp.Responder {
	args := []resp.ResponderOptFn{
		resp.WithRootUrl(url.String()),
		resp.WithLogger(l),
		resp.WithParser(p),
		resp.WithAuthTemplate(defaultAuthedTmpl),
		resp.WithErrTemplate(defaultErrTmpl),
		resp.WithUnauthTemplate(defaultUnauthedTmpl),
		resp.WithVueTemplate(defaultVueTmpl),
	}

	if len(ctxKeys) > 0 {
		args = append(args, resp.WithCtxKeys(ctxKeys...))
	}

	return resp.NewResponder(args...)
}

// defaultRouter constructs a router.Router to be used by the web server.
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
// - "APP_TITLE"
// - "SESSION_AUTH_KEY"
// - "SESSION_ENCRYPTION_KEY"
//
// Both KEY env vars be valid hex encoded values.
func defaultSessionStore(env trails.Environment) (session.SessionStorer, error) {
	appName := os.Getenv(appTitleEnvVar)
	if appName == "" {
		return nil, fmt.Errorf("%w: APP_TITLE cannot be unset, got %q", trails.ErrBadConfig, appName)
	}

	appName = cases.Lower(language.English).String(appName)
	appName = regexp.MustCompile(`[,':]`).ReplaceAllString(appName, "")
	appName = regexp.MustCompile(`\s`).ReplaceAllString(appName, "-")

	cfg := session.Config{
		AuthKey:     os.Getenv(sessionAuthKeyEnvVar),
		EncryptKey:  os.Getenv(sessionEncryptKeyEnvVar),
		Env:         env,
		SessionName: "trails-" + appName,
	}

	args := []session.ServiceOpt{
		session.WithCookie(),
		session.WithMaxAge(3600 * 24 * 7),
	}

	return session.NewStoreService(cfg, args...)
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
