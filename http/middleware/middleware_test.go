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

func newTestUserStore(b bool) middleware.UserStorer {
	return func(_ uint) (middleware.User, error) {
		return testUser(b), nil
	}
}

func newFailedUserStore(b bool) middleware.UserStorer {
	return func(_ uint) (middleware.User, error) {
		return testUser(b), errors.New("")
	}
}

func teapotHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
}

func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

type testLogger struct {
	*bytes.Buffer
}

func newLogger() testLogger { return testLogger{new(bytes.Buffer)} }

func (tl testLogger) AddSkip(i int) logger.Logger            { return tl }
func (tl testLogger) Skip() int                              { return 0 }
func (tl testLogger) Debug(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Error(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Fatal(msg string, _ *logger.LogContext) { fmt.Fprint(tl, msg) }
func (tl testLogger) Info(msg string, _ *logger.LogContext)  { fmt.Fprint(tl, msg) }
func (tl testLogger) Warn(msg string, _ *logger.LogContext)  { fmt.Fprint(tl, msg) }
