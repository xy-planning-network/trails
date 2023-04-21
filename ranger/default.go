package ranger

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	BaseURLEnvVar = "BASE_URL"

	// App metadata
	AppDescEnvVar    = "APP_DESCRIPTION"
	AppTitleEnvVar   = "APP_TITLE"
	ContactUsEnvVar  = "CONTACT_US_EMAIL"
	defaultContactUs = "hello@xyplanningnetwork.com"

	// Environment defaults
	environmentEnvVar = "ENVIRONMENT"

	// Log defaults
	logLevelEnvVar = "LOG_LEVEL"
	defaultLogLvl  = logger.LogLevelInfo

	// Database defaults
	dbHostEnvVar     = "DATABASE_HOST"
	defaultDBHost    = "localhost"
	dbNameEnvVar     = "DATABASE_NAME"
	dbPassEnvVar     = "DATABASE_PASSWORD"
	dbPortEnvVar     = "DATABASE_PORT"
	defaultDBPort    = "5432"
	dbSSLModeEnvVar  = "DATABASE_SSLMODE"
	defaultDBSSLMode = "prefer"
	dbURLEnvVar      = "DATABASE_URL"
	dbUserEnvVar     = "DATABASE_USER"

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

	return cfg
}

// defaultDB connects to a Postgres database
// using default configuration environment variables
// and runs the list of [postgres.Migration] passed in.
func defaultDB(env trails.Environment, list []postgres.Migration) (postgres.DatabaseService, error) {
	db, err := postgres.Connect(NewPostgresConfig(env), list)
	if err != nil {
		return nil, err
	}

	return postgres.NewService(db), nil
}

// defaultLogger constructs a [logger.Logger].
// A default logger.Logger can be overriden for in unit tests.
func defaultLogger(env trails.Environment, output io.Writer) logger.Logger {
	if output == nil {
		output = os.Stdout
	}

	out := log.New(output, "", log.LstdFlags)
	logLvl := trails.EnvVarOrLogLevel(logLevelEnvVar, defaultLogLvl)
	args := []logger.LoggerOptFn{
		logger.WithEnv(env.String()),
		logger.WithLevel(logLvl),
		logger.WithLogger(out),
	}

	return logger.New(args...)
}

// defaultParser constructs a template.Parser to be used
// when responding to HTTP requests with [*http/resp.Responder.Html].
//
// defaultParser makes available these functions in an HTML template:
//
//   - "env"
//   - "metadata"
//     - "description" returns the value set by the APP_DESCRIPTION env var
//     - "title" returns the value set by the APP_TITLE env var
//   - "nonce"
//   - "rootUrl"
//   - "packTag"
//   - "isDevelopment"
//   - "isStaging"
//   - "isProduction"
func defaultParser(env trails.Environment, url *url.URL, files fs.FS, m Metadata) template.Parser {
	args := []template.ParserOptFn{
		template.WithFn(template.Env(env)),
		template.WithFn("isDevelopment", env.IsDevelopment),
		template.WithFn("isStaging", env.IsStaging),
		template.WithFn("isProduction", env.IsProduction),
		template.WithFn(m.templateFunc()),
		template.WithFn(template.Nonce()),
		template.WithFn("packTag", template.TagPacker(env, os.DirFS("."))),
		template.WithFn(template.RootUrl(url)),
	}

	return template.NewParser([]fs.FS{files, tmpls}, args...)
}

// defaultResponder configures the [*resp.Responder] to be used by http.Handlers.
func defaultResponder(l logger.Logger, url *url.URL, p template.Parser, contact string) *resp.Responder {
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
