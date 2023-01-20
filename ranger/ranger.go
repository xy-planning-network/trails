package ranger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
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
	*resp.Responder
	router.Router

	cancel    context.CancelFunc
	ctx       context.Context
	db        postgres.DatabaseService
	env       trails.Environment
	kr        keyring.Keyringable
	l         logger.Logger
	shutdowns []ShutdownFn
	srv       *http.Server
}

// New constructs a Ranger from the provided options.
// Default options are applied first followed by the options passed into New.
// Options supplied to New overwrite default configurations.
func New[User RangerUser](cfg Config[U]) (*Ranger, error) {
	err := cfg.Valid()
	if err != nil {
		return nil, err
	}

	r := new(Ranger)

	// Setup initial configuration
	r.env = trails.EnvVarOrEnv(environmentEnvVar, trails.Development)
	r.l = defaultLogger()
	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.db, err = defaultDB(r.env, c.Migrations)
	if err != nil {
		return nil, err
	}

	url := trails.EnvVarOrURL(baseURLEnvVar, defaultBaseURL)
	r.Responder = defaultResponder(r.l, url, defaultParser(r.env, url, cfg.FS), cfg.Keyring)

	sess, err := defaultSessionStore(r.env, r.kr)
	if err != nil {
		return nil, err
	}

	r.userstore = cfg.defaultUserStore(r.db)
	mws := make([]middleware.Adapter, 0)
	// NOTE(dlk): PRODUCTION only middlewares
	if r.env.IsProduction() {
		mws = append(
			mws,
			middleware.ForceHTTPS(r.env),
		)
	}

	mws = append(
		mws,
		middleware.RequestID(r.kr.Key("RequestID")),
		middleware.InjectIPAddress(),
		middleware.LogRequest(r.l),
		middleware.InjectSession(sess, r.kr.SessionKey()),
		middleware.CurrentUser(r.Responder, userstore, r.kr.SessionKey(), r.kr.CurrentUserKey()),
	)
	r.Router = defaultRouter(r.env, url, r.Responder, mws)
	r.srv = defaultServer(r.ctx)

	return r, nil
}

func (r *Ranger) CancelContext()               { r.cancel() }
func (r *Ranger) DB() postgres.DatabaseService { return r.db }
func (r *Ranger) Env() trails.Environment      { return r.env }
func (r *Ranger) Logger() logger.Logger        { return r.l }

// Guide begins the web server.
//
// These, and (*Ranger).Shutdown, stop Guide:
//
// - os.Interrupt
// - os.Kill
// - syscall.SIGHUP
// - syscall.SIGINT
// - syscall.SIGQUIT
// - syscall.SIGTERM
func (r *Ranger) Guide() error {
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

	cc := logger.CurrentCaller()
	go func() {
		s := <-ch
		r.l.Info(fmt.Sprint("received shutdown signal: ", s), &logger.LogContext{Caller: cc})
		r.cancel()
	}()

	go func() {
		r.l.Info(fmt.Sprintf("running web server at %s", r.srv.Addr), &logger.LogContext{Caller: cc})
		r.srv.Handler = r.Router
		if err := r.srv.ListenAndServe(); err != http.ErrServerClosed {
			err = fmt.Errorf("could not listen: %w", err)
			r.l.Error(err.Error(), &logger.LogContext{Caller: cc})
		}
	}()

	<-r.ctx.Done()
	close(ch)

	return r.Shutdown()
}

// Shutdown shutdowns the web server.
//
// If you pass custom ShutdownFns using WithShutdowns,
// Shutdown calls these before closing the web server.
//
// You may want to provide custom ShutdownFns if other services
// ought to be stopped before the web server stops accepts requests.
//
// In such a case, Ranger continues to accept HTTP requests
// until these custom ShutdownFns finish.
// This state of affairs ought to be gracefully handled in your web handlers.
func (r *Ranger) Shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ll := r.l
	if sl, ok := ll.(logger.SkipLogger); ok {
		ll = sl.AddSkip(sl.Skip() + 2)
	}

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

type ShutdownFn func(context.Context) error
