package ranger

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

// A RangerUser is the kind of functionality an application's User must fulfill
// in order to take advantage of trails.
//
// NOTE(dlk): refer to this example as to why we have all the theatrics around generics:
// https://go.dev/play/p/IfXLlgaJUM_N
type RangerUser interface {
	middleware.User
}

// A Ranger manages and exposes all components of a trails app to one another.
type Ranger struct {
	logger.Logger
	*resp.Responder
	router.Router

	assetsURL  *url.URL
	cancel     context.CancelFunc
	ctx        context.Context
	db         postgres.DatabaseService
	env        trails.Environment
	metadata   Metadata
	migrations []postgres.Migration
	sessions   session.SessionStorer
	shutdowns  []ShutdownFn
	srv        *http.Server
	url        *url.URL
}

// New constructs a Ranger from the provided options.
// Default options are applied first followed by the options passed into New.
// Options supplied to New overwrite default configurations.
func New[U RangerUser](cfg Config[U]) (*Ranger, error) {
	err := cfg.Valid()
	if err != nil {
		return nil, err
	}

	if cfg.logoutput == nil {
		cfg.logoutput = os.Stdout
	}

	r := new(Ranger)

	// Setup initial configuration
	r.env = trails.EnvVarOrEnv(environmentEnvVar, trails.Development)
	r.Logger = defaultAppLogger(r.env, cfg.logoutput)
	if _, ok := r.Logger.(*logger.SentryLogger); ok {
		r.shutdowns = append(r.shutdowns, logger.FlushSentry)
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.assetsURL = trails.EnvVarOrURL(AssetsURLEnvVar, defaultAssetsURL)
	r.url = trails.EnvVarOrURL(BaseURLEnvVar, defaultBaseURL)
	r.metadata, err = newMetadata()
	if err != nil {
		return nil, err
	}

	// Return minimal ranger with configured routing for maintenance mode
	if cfg.MaintMode {
		return newMaintRanger(r, cfg), nil
	}

	r.migrations = cfg.Migrations
	if cfg.mockdb == nil {
		r.db, err = defaultDB(r.env)
		if err != nil {
			return nil, err
		}
	} else {
		r.db = cfg.mockdb
	}

	r.Responder = defaultResponder(r.Logger, r.url, defaultParser(r.env, r.url, r.assetsURL, cfg.FS, r.metadata), r.metadata.Contact)

	r.sessions, err = defaultSessionStore(r.env, r.metadata.Title)
	if err != nil {
		return nil, err
	}

	userstore := cfg.defaultUserStore(r.db)
	var mws []middleware.Adapter
	// NOTE(dlk): PRODUCTION only middlewares
	if r.env.IsProduction() {
		mws = append(
			mws,
			middleware.ForceHTTPS(r.env),
		)
	}

	logReq := middleware.LogRequest(defaultHTTPLogger(r.env, cfg.logoutput))

	mws = append(
		mws,
		logReq,
		middleware.RequestID(),
		middleware.InjectIPAddress(),
		middleware.InjectSession(r.sessions),
		middleware.CurrentUser(r.Responder, userstore),
	)
	r.Router = defaultRouter(r.env, r.url, r.Responder, logReq, mws)
	r.srv = defaultServer(r.ctx)

	return r, nil
}

func (r *Ranger) AssetsURL() *url.URL                            { return r.assetsURL }
func (r *Ranger) BaseURL() *url.URL                              { return r.url }
func (r *Ranger) Context() (context.Context, context.CancelFunc) { return r.ctx, r.cancel }
func (r *Ranger) DB() postgres.DatabaseService                   { return r.db }
func (r *Ranger) Env() trails.Environment                        { return r.env }
func (r *Ranger) Metadata() Metadata                             { return r.metadata }
func (r *Ranger) SessionStore() session.SessionStorer            { return r.sessions }

// Guide begins the web server.
//
// These, and (*Ranger).Shutdown, stop Guide:
//   - os.Interrupt
//   - syscall.SIGHUP
//   - syscall.SIGINT
//   - syscall.SIGQUIT
//   - syscall.SIGTERM
func (r *Ranger) Guide() error {
	// NOTE(dlk): check the concrete type as it may be the desired type
	// or *postgres.MockDatabaseService,
	// which we don't need to run migrations against.
	if db, ok := r.db.(*postgres.DatabaseServiceImpl); ok {
		if err := postgres.MigrateUp(db.DB, r.migrations); err != nil {
			return err
		}
	}

	if r.ctx == nil {
		r.ctx, r.cancel = context.WithCancel(context.Background())
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(
		ch,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	pc, _, _, _ := runtime.Caller(1)
	go func() {
		s := <-ch
		r.Info(fmt.Sprint("received shutdown signal: ", s), &logger.LogContext{Caller: pc})
		r.cancel()
	}()

	go func() {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					r.Info(fmt.Sprintf("running %s on commit: %s", info.GoVersion, setting.Value), &logger.LogContext{Caller: pc})
					break
				}
			}
		}

		r.Info(fmt.Sprintf("running web server at %s", r.srv.Addr), &logger.LogContext{Caller: pc})
		r.srv.Handler = r.Router
		if err := r.srv.ListenAndServe(); err != http.ErrServerClosed {
			err = fmt.Errorf("could not listen: %w", err)
			r.Error(err.Error(), nil)
		}
	}()

	<-r.ctx.Done()
	close(ch)

	return r.shutdown()
}

// Shutdown shutdowns the web server
// and cancels the context.Context exposed by *Ranger.Context.
//
// If you pass custom ShutdownFns using Config.Shutdowns,
// Shutdown calls these before closing the web server.
//
// You may want to provide custom ShutdownFns if other services
// ought to be stopped before the web server stops accepts requests.
//
// In such a case, Ranger continues to accept HTTP requests
// until these custom ShutdownFns finish.
// This state of affairs ought to be gracefully handled in your web handlers.
func (r *Ranger) Shutdown() {
	// NOTE(dlk): this misdirection exists to ensure any dependencies on this *Ranger
	// not using a ShutdownFn can clean themselves up,
	// given *Ranger has been told to shutdown.
	r.cancel()
}

func (r *Ranger) shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ll := r.Logger.AddSkip(r.Logger.Skip() + 2)

	ll.Info("shutting down web server", nil)
	if len(r.shutdowns) > 0 {
		ll.Info("shutting down plugins", nil)
		for _, fn := range r.shutdowns {
			if err := fn(shutdownCtx); err != nil {
				ll.Error("failed shutting down: "+err.Error(), nil)
				return err
			}
		}
	}

	err := r.srv.Shutdown(shutdownCtx)
	if err == http.ErrServerClosed {
		ll.Info("web server shutdown successfully", nil)
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not shutdown: %w", err)
	}

	ll.Info("web server shutdown successfully", nil)
	return nil
}

// BuildWorkerCore constructs a *Ranger but skips those components relating to the HTTP router.
func BuildWorkerCore() (*Ranger, error) {
	var err error
	r := new(Ranger)
	r.env = trails.EnvVarOrEnv(environmentEnvVar, trails.Development)
	r.Logger = defaultWorkerLogger(r.env)

	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.db, err = defaultDB(r.env)
	if err != nil {
		return nil, err
	}

	r.url = trails.EnvVarOrURL(BaseURLEnvVar, defaultBaseURL)
	r.metadata, err = newMetadata()
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Metadata captures values set by different env vars
// used to customize identifying the application to end users.
//
// Metadata provides its data through the "metadata" template function.
type Metadata struct {
	Contact string
	Desc    string
	Title   string
}

func newMetadata() (Metadata, error) {
	m := Metadata{
		Contact: trails.EnvVarOrString(ContactUsEnvVar, defaultContactUs),
		Desc:    os.Getenv(AppDescEnvVar),
		Title:   os.Getenv(AppTitleEnvVar),
	}

	if m.Contact == "" {
		err := fmt.Errorf("%w: missing %q", trails.ErrBadConfig, ContactUsEnvVar)
		return Metadata{}, err

	}

	if m.Desc == "" {
		err := fmt.Errorf("%w: missing %q", trails.ErrBadConfig, AppDescEnvVar)
		return Metadata{}, err
	}

	if m.Title == "" {
		err := fmt.Errorf("%w: missing %q", trails.ErrBadConfig, AppTitleEnvVar)
		return Metadata{}, err
	}

	return m, nil
}

func (m Metadata) templateFunc() (string, func(key string) string) {
	return "metadata", func(key string) string {
		return map[string]string{
			"contactUs":   m.Contact,
			"description": m.Desc,
			"title":       m.Title,
		}[key]
	}
}

type ShutdownFn func(context.Context) error

// newMaintRanger configures the bare minimum to render an HTML maintenance page.
// This includes logging.
func newMaintRanger[U RangerUser](r *Ranger, cfg Config[U]) *Ranger {
	logReq := middleware.LogRequest(defaultHTTPLogger(r.env, cfg.logoutput))
	mws := []middleware.Adapter{
		middleware.RequestID(),
		middleware.InjectIPAddress(),
		logReq,
	}

	r.Router = router.New(r.env.String(), logReq)
	r.Router.OnEveryRequest(mws...)

	r.Router.CatchAll(MaintModeHandler(
		defaultParser(r.env, r.url, r.assetsURL, cfg.FS, r.metadata),
		r.Logger,
		r.metadata.Contact),
	)

	r.srv = defaultServer(r.ctx)

	r.Logger.Info("Maintenance mode is turned on", nil)

	return r
}

func MaintModeHandler(p *template.Parser, l logger.Logger, contact string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Retry-After", "600")
		w.WriteHeader(http.StatusServiceUnavailable)

		tmpl, err := p.Parse("tmpl/maintenance.tmpl")
		if err != nil {
			l.Error(err.Error(), nil)
			return
		}
		if err := tmpl.Execute(w, contact); err != nil {
			l.Error(err.Error(), nil)
			return
		}
	}
}
