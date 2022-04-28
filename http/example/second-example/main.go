package main

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/xy-planning-network/trails/http/template"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/ranger"
)

//go:embed tmpl/*
var files embed.FS

type handler struct {
	*ranger.Ranger
}

func (h *handler) root(w http.ResponseWriter, r *http.Request) {

}

func itsHammerTime() string {
	return time.Now().Format("Hammer time is now: 2006-01-02 13:04:05")
}

func main() {
	// Let's customize how we log in our app
	// by bringing our own implementation of logger.Logger.
	l := NewPingPonger()

	// Let's customize how we render HTML templates.
	// Instead of starting with template.New, this piggybacks off the defaults
	// add a custom function our templates can leverage.
	//
	p := ranger.DefaultParser(files, template.WithFn("hammerTime", itsHammerTime))

	// add custom components to the constructing so they override defaults.
	rng, err := ranger.New(ranger.WithLogger(l), p)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := rng.Guide(); err != nil {
		fmt.Println(err)
		return
	}
}

type pingPonger struct {
	i int
	l logger.Logger
}

func NewPingPonger() pingPonger {
	return pingPonger{0, logger.NewLogger()}
}

func (p pingPonger) pingpong() string {
	if p.i%2 == 0 {
		return "ping: "
	}

	return "pong: "
}

func (p pingPonger) Debug(msg string, ctx *logger.LogContext) {
	p.l.Debug(p.pingpong()+msg, ctx)
}
func (p pingPonger) Error(msg string, ctx *logger.LogContext) {
	p.l.Error(p.pingpong()+msg, ctx)
}
func (p pingPonger) Fatal(msg string, ctx *logger.LogContext) {
	p.l.Fatal(p.pingpong()+msg, ctx)
}
func (p pingPonger) Info(msg string, ctx *logger.LogContext) {
	p.l.Info(p.pingpong()+msg, ctx)
}
func (p pingPonger) Warn(msg string, ctx *logger.LogContext) {
	p.l.Warn(p.pingpong()+msg, ctx)
}
func (p pingPonger) LogLevel() logger.LogLevel { return p.l.LogLevel() }
