package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"sync"
	"time"

	. "github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/router"
	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/ranger"
)

//go:embed tmpl/*
var files embed.FS

const (
	dir   = "tmpl"
	first = dir + "/first.tmpl"
)

type handler struct {
	*ranger.Ranger
}

// root is a fully-formed use of Responder.
func (h handler) root(w http.ResponseWriter, r *http.Request) {
	if err := h.Html(w, r, Unauthed(), Tmpls(first)); err != nil {
		h.Err(w, r, err)
	}
}

// initShutdown uses a closure to inject dependencies to our http.Handler,
// showing an alternative pattern to using a struct to accomplish this requirement.
//
// Requesting the endpoint the enclosed function binds to causes the web server
// to shutdown!
//
// As well, this handler continues to use http.ResponseWriter's own methods
// for writing to the client.
// Unless there's functionality in Responder we need,
// no reason to not use the std lib!
func initShutdown(h handler, cancel context.CancelFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Debug("see ya!", nil)
		w.WriteHeader(http.StatusOK)
		r.Body.Close()
		cancel()
	})
}

// itsHammerTime is a template function saying when hammer time is.
func itsHammerTime() string {
	return time.Now().Format("Hammer time is now: 2006-01-02 03:04:05")
}

func main() {
	// Let's setup an application context to be passed to Ranger,
	// we'll have some fun with this in the http.Handler returned by initShutdown.
	ctx, cancel := context.WithCancel(context.Background())

	// Let's customize how we log in our app
	// by bringing our own implementation of logger.Logger.
	l := NewPingPonger()

	// Let's customize how we render HTML templates.
	// Instead of starting with template.New, this piggybacks off the defaults
	// and adds a custom function our templates can call.
	p := ranger.DefaultParser(files, template.WithFn("hammerTime", itsHammerTime))

	// Add custom components to the constructing so they override defaults.
	//
	// Notably, this Ranger does not utilize sessions.
	// Starting the web server will warn us of this fact,
	// but start up anyways and being accepted requests.
	rng, err := ranger.New(
		ranger.WithContext(ctx),
		ranger.WithLogger(l),

		// TODO(dlk):
		// to prevent fetching a session key from the keyring, we have to pass in a nil value.
		// Instead consider removing SessionKey & CurrentUserKey from keyring.Keyringable
		// and implement a simpler version in http/session that adds these methods.
		ranger.WithKeyring(nil),
		p,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	h := handler{rng}
	rng.HandleRoutes([]router.Route{
		{Path: "/", Method: http.MethodGet, Handler: h.root},
		{Path: "/shutdown", Method: http.MethodGet, Handler: initShutdown(h, cancel)},
	})

	if err := rng.Guide(); err != nil {
		fmt.Println(err)
		return
	}
}

// A pingPonger logs messages while prepending "ping" or "pong" before it.
//
// TODO(dlk): check SkipLogger
type pingPonger struct {
	i  int
	l  logger.Logger
	mu sync.Mutex
}

func NewPingPonger() *pingPonger {
	return &pingPonger{0, logger.New(), sync.Mutex{}}
}

func (p *pingPonger) pingpong() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.i++
	if p.i%2 == 0 {
		return "pong: "
	}

	return "ping: "
}

func (p *pingPonger) Debug(msg string, ctx *logger.LogContext) {
	if p.l.LogLevel() > logger.LogLevelDebug {
		return
	}
	p.l.Debug(p.pingpong()+msg, ctx)
}
func (p *pingPonger) Error(msg string, ctx *logger.LogContext) {
	if p.l.LogLevel() > logger.LogLevelError {
		return
	}
	p.l.Error(p.pingpong()+msg, ctx)
}
func (p *pingPonger) Fatal(msg string, ctx *logger.LogContext) {
	if p.l.LogLevel() > logger.LogLevelFatal {
		return
	}
	p.l.Fatal(p.pingpong()+msg, ctx)
}
func (p *pingPonger) Info(msg string, ctx *logger.LogContext) {
	if p.l.LogLevel() > logger.LogLevelInfo {
		return
	}
	p.l.Info(p.pingpong()+msg, ctx)
}
func (p *pingPonger) Warn(msg string, ctx *logger.LogContext) {
	if p.l.LogLevel() > logger.LogLevelWarn {
		return
	}
	p.l.Warn(p.pingpong()+msg, ctx)
}
func (p *pingPonger) LogLevel() logger.LogLevel { return p.l.LogLevel() }
