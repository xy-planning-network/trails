package ranger

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	// TODO(dlk): configurable env files
	_ "github.com/joho/godotenv/autoload"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

// A Ranger manages and exposes all components of a trails app to one another.
type Ranger struct {
	*resp.Responder
	router.Router

	ctx      context.Context
	db       postgres.DatabaseService
	env      Environment
	kr       keyring.Keyringable
	l        logger.Logger
	p        template.Parser
	sessions session.SessionStorer
	srv      *http.Server
	url      *url.URL
}

// New constructs a Ranger from the provided options.
// Default options are applied first followed by the options passed into New.
// Options supplied to New overwrite default configurations.
func New(opts ...RangerOption) (*Ranger, error) {
	r := new(Ranger)
	followups := make([]OptFollowup, 0)

	// NOTE(dlk): calling an option configures the *Ranger under construction.
	// Some options require data from other options.
	// These options, therefore, must delay configuring the *Ranger
	// until either (1) user supplied RangerOptions or (2) default RangerOptions
	// configure the *Ranger first.
	// They return an optFollowup to be called after the initial set of options are run.
	for _, opt := range append(defaultOpts(), opts...) {
		fn, err := opt(r)
		if err != nil {
			return r, fmt.Errorf("%w: %s", ErrBadConfig, err)
		}

		if fn != nil {
			followups = append(followups, fn)
		}
	}

	for _, fn := range followups {
		if err := fn(); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBadConfig, err)
		}
	}

	r.p = nil

	return r, nil
}

func (r *Ranger) EmitDB() postgres.DatabaseService        { return r.db }
func (r *Ranger) EmitKeyring() keyring.Keyringable        { return r.kr }
func (r *Ranger) EmitLogger() logger.Logger               { return r.l }
func (r *Ranger) EmitSessionStore() session.SessionStorer { return r.sessions }

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
	var cancel context.CancelFunc
	r.ctx, cancel = context.WithCancel(context.Background())

	ch := make(chan os.Signal, 1)
	signal.Notify(
		ch,
		os.Interrupt,
		os.Kill,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func() {
		s := <-ch
		r.l.Info(fmt.Sprint("received shutdown signal: ", s), nil)
		cancel()
	}()

	go func() {
		r.l.Info(fmt.Sprintf("running web server at %s", r.srv.Addr), nil)
		r.srv.Handler = r.Router
		if err := r.srv.ListenAndServe(); err != http.ErrServerClosed {
			err = fmt.Errorf("could not listen: %w", err)
			r.l.Error(err.Error(), nil)
		}
	}()

	<-r.ctx.Done()
	return r.Shutdown()
}

// Shutdown shutdowns the web server.
func (r *Ranger) Shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r.l.Info("shutting down web server", nil)
	err := r.srv.Shutdown(shutdownCtx)
	if err == http.ErrServerClosed {
		r.l.Info("web server shutdown successfully", nil)
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not shutdown: %w", err)
	}

	r.l.Info("web server shutdown successfully", nil)
	return nil
}
