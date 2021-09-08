/*

middleware_test utilizes build tags to exclude tests for regular dev workflows.

Include a -tags $TAG_NAME flag to include otherwise excluded tests.

*/
package middleware_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
)

var (
	_ middleware.User = testUser(true)
)

type testUser bool

func (b testUser) HasAccess() bool { return bool(b) }
func (testUser) HomePath() string  { return "/" }
func (testUser) GetEmail() string  { return "user@example.com" }
func (testUser) GetID() uint       { return 1 }

type testUserStore testUser

func (s testUserStore) GetByID(_ uint) (middleware.User, error) { return testUser(s), nil }

type failedUserStore testUser

func (s failedUserStore) GetByID(_ uint) (middleware.User, error) { return testUser(s), errors.New("") }

func teapotHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
}

func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

type ctxKey string

func (k ctxKey) Key() string    { return string(k) }
func (k ctxKey) String() string { return string(k) }

type failedSession struct {
	e error
}

func (failedSession) ClearFlashes(w http.ResponseWriter, r *http.Request) {}
func (e failedSession) Flashes(w http.ResponseWriter, r *http.Request) []session.Flash {
	return nil
}
func (e failedSession) SetFlash(w http.ResponseWriter, r *http.Request, flash session.Flash) error {
	return error(e.e)
}
func (e failedSession) Delete(w http.ResponseWriter, r *http.Request) error      { return error(e.e) }
func (e failedSession) Get(key string) interface{}                               { return nil }
func (e failedSession) ResetExpiry(w http.ResponseWriter, r *http.Request) error { return error(e.e) }
func (e failedSession) Save(w http.ResponseWriter, r *http.Request) error        { return error(e.e) }
func (e failedSession) Set(w http.ResponseWriter, r *http.Request, key string, val interface{}) error {
	return error(e.e)
}
func (e failedSession) DeregisterUser(w http.ResponseWriter, r *http.Request) error {
	return error(e.e)
}
func (e failedSession) RegisterUser(w http.ResponseWriter, r *http.Request, ID uint) error {
	return error(e.e)
}
func (e failedSession) UserID() (uint, error) { return 0, error(e.e) }

type testLogger struct {
	*bytes.Buffer
}

func newLogger() testLogger { return testLogger{new(bytes.Buffer)} }

func (tl testLogger) Debug(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Error(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Fatal(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Info(msg string, _ *logger.LogContext)  { fmt.Fprint(tl, msg) }
func (tl testLogger) Warn(msg string, _ *logger.LogContext)  { fmt.Fprint(tl, msg) }
func (tl testLogger) LogLevel() logger.LogLevel              { return logger.LogLevelDebug }
