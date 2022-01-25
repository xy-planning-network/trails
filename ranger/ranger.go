package ranger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// TODO(dlk): configurable env files
	_ "github.com/joho/godotenv/autoload"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

// A Ranger manages and exposes all components of a trails app to one another.
type Ranger struct {
	DB      postgres.DatabaseService
	Env     Environment
	Keyring keyring.Keyringable
	logger.Logger
	*resp.Responder
	Router router.Router

	ctx  context.Context
	p    template.Parser
	sess session.SessionStorer
	srv  *http.Server
}

// NewRanger constructs a Ranger from the provided options.
// Default options are applied first followed by the options passed into NewRanger.
// Options supplied to NewRanger overwrite default configurations.
func NewRanger(opts ...RangerOption) (*Ranger, error) {
	r := new(Ranger)
	followups := make([]optFollowup, 0)

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

	return r, nil
}

// TODO(dlk)
//
// Implements middleware.UserStorer.
func (r *Ranger) GetByID(id uint) (middleware.User, error) {
	var model interface{}
	if err := r.DB.FindByID(model, id); err != nil {
		return nil, err
	}

	return model.(middleware.User), nil
}

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
		r.Info(fmt.Sprint("received shutdown signal: ", s), nil)
		cancel()
	}()

	go func() {
		r.Info(fmt.Sprintf("running web server at %s", r.srv.Addr), nil)
		r.srv.Handler = r.Router
		if err := r.srv.ListenAndServe(); err != http.ErrServerClosed {
			err = fmt.Errorf("could not listen: %w", err)
			r.Error(err.Error(), nil)
		}
	}()

	<-r.ctx.Done()
	return r.Shutdown()
}

// Shutdown shutdowns the web server.
func (r *Ranger) Shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r.Info("shutting down web server", nil)
	err := r.srv.Shutdown(shutdownCtx)
	if err == http.ErrServerClosed {
		r.Info("web server shutdown successfully", nil)
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not shutdown: %w", err)
	}

	r.Info("web server shutdown successfully", nil)
	return nil
}