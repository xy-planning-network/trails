package resp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
	"github.com/xy-planning-network/trails/logger"
)

type testFn func(*testing.T, *httptest.ResponseRecorder, *http.Request, error)

const (
	jsonMediaType = "application/json; charset=UTF-8"
)

func TestResponderDo(t *testing.T) {
	t.Run("Cancelled", func(t *testing.T) {
		// Arrange
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx, cancel := context.WithCancel(r.Context())
		r = r.Clone(ctx)

		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusPaymentRequired)

		cancel()

		d := resp.NewResponder()

		// Act
		err := d.Json(w, r, resp.Code(http.StatusTeapot))

		// Assert
		require.ErrorIs(t, err, resp.ErrDone)
		require.Equal(t, http.StatusPaymentRequired, w.Code)
	})
}

func TestResponderCurrentUser(t *testing.T) {
	tcs := []struct {
		name        string
		ctx         context.Context
		expectedVal any
		expectedErr error
	}{
		{"Not-Set", context.Background(), nil, resp.ErrNotFound},
		{
			"Wrong-Key",
			context.WithValue(context.Background(), trails.Key("not-current-user-key"), struct{}{}),
			nil,
			resp.ErrNotFound,
		},
		{
			"Set-With-Nil",
			context.WithValue(context.Background(), trails.CurrentUserKey, nil),
			nil,
			resp.ErrNotFound,
		},
		{
			"Set-With-Val",
			context.WithValue(context.Background(), trails.CurrentUserKey, struct{}{}),
			struct{}{},
			nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := resp.NewResponder()
			actual, err := d.CurrentUser(tc.ctx)
			require.ErrorIs(t, err, tc.expectedErr)
			require.Equal(t, tc.expectedVal, actual)
		})
	}
}

func TestResponderErr(t *testing.T) {
	tcs := []struct {
		name     string
		expected error
	}{
		{"Nil", nil},
		{"ErrDone", resp.ErrDone},
		{"Custom", errors.New("my favorite error")},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			w := httptest.NewRecorder()
			l := newLogger()
			d := resp.NewResponder(resp.WithLogger(l))

			// Act
			d.Err(w, r, tc.expected)

			// Assert
			require.Equal(t, http.StatusInternalServerError, w.Code)
			if tc.expected != nil {
				require.Equal(t, tc.expected.Error(), l.b.String())
			}
		})
	}
}

func TestResponderJson(t *testing.T) {
	tcs := []struct {
		name   string
		fns    []resp.Fn
		assert testFn
	}{
		{
			name: "Zero-Value",
			fns:  []resp.Fn{},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, jsonMediaType, w.Header().Get("Content-Type"))
				require.Equal(t, []byte("{}\n"), w.Body.Bytes())
			},
		},
		{
			name: "With-Code",
			fns:  []resp.Fn{resp.Code(http.StatusTeapot)},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTeapot, w.Code)
				require.Equal(t, jsonMediaType, w.Header().Get("Content-Type"))
				require.Equal(t, []byte("{}\n"), w.Body.Bytes())
			},
		},
		{
			name: "With-Data",
			fns:  []resp.Fn{resp.Data(map[string]any{"go": "rocks"})},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, jsonMediaType, w.Header().Get("Content-Type"))

				var b bytes.Buffer
				err = json.NewEncoder(&b).Encode(map[string]map[string]string{"data": {"go": "rocks"}})
				require.Nil(t, err)
				require.Equal(t, b.Bytes(), w.Body.Bytes())
			},
		},
		{
			name: "With-User",
			fns:  []resp.Fn{resp.CurrentUser(1)},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, jsonMediaType, w.Header().Get("Content-Type"))

				var b bytes.Buffer
				err = json.NewEncoder(&b).Encode(map[string]int{"currentUser": 1})
				require.Nil(t, err)
				require.Equal(t, b.Bytes(), w.Body.Bytes())
			},
		},
		{
			name: "With-Code-Data-User",
			fns: []resp.Fn{
				resp.Code(http.StatusTeapot),
				resp.CurrentUser(1),
				resp.Data(map[string]any{"go": "rocks"}),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTeapot, w.Code)
				require.Equal(t, jsonMediaType, w.Header().Get("Content-Type"))

				var b bytes.Buffer
				err = json.NewEncoder(&b).
					Encode(
						struct {
							D any `json:"data"`
						}{
							D: map[string]string{"go": "rocks"},
						},
					)
				require.Nil(t, err)
				require.Equal(t, b.Bytes(), w.Body.Bytes())
			},
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		w := httptest.NewRecorder()
		d := resp.NewResponder()
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, w, r, d.Json(w, r, tc.fns...))
		})
	}
}

func TestResponderRaw(t *testing.T) {
	// TODO?
}

func TestResponderRedirect(t *testing.T) {
	otherURL := "http://otherexample.com"
	tcs := []struct {
		name   string
		fns    []resp.Fn
		assert testFn
	}{
		{
			name: "No-Fns",
			fns:  []resp.Fn{},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrMissingData)
			},
		},
		{
			name: "Param-No-Url",
			fns: []resp.Fn{
				resp.Params(map[string]string{"test": "true"}),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrMissingData)
			},
		},
		{
			name: "Params4x-Url-Redirect",
			fns: []resp.Fn{
				resp.Params(map[string]string{"test": "true"}),
				resp.Params(map[string]string{"go": "fun"}),
				resp.Params(map[string]string{"params": "4"}),
				resp.Params(map[string]string{"good": "times"}),
				resp.Url(otherURL + "/redirect"),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusFound, w.Code)

				expected, err := url.ParseRequestURI(otherURL + "/redirect")
				require.Nil(t, err)

				q := expected.Query()
				q.Add("test", "true")
				q.Add("go", "fun")
				q.Add("params", "4")
				q.Add("good", "times")
				expected.RawQuery = q.Encode()

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)
				require.Equal(t, expected.String(), actual.String())
				require.Equal(t, expected.Query(), actual.Query())
			},
		},
		{
			name: "Overwrite-4xx",
			fns: []resp.Fn{
				resp.Url(otherURL),
				resp.Code(http.StatusTeapot),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusSeeOther, w.Code)

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)

				expected, err := url.ParseRequestURI(otherURL)
				require.Nil(t, err)
				require.Equal(t, expected.String(), actual.String())
			},
		},
		{
			name: "Overwrite-5xx",
			fns: []resp.Fn{
				resp.Url(otherURL),
				resp.Code(http.StatusInsufficientStorage),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTemporaryRedirect, w.Code)

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)

				expected, err := url.ParseRequestURI(otherURL)
				require.Nil(t, err)
				require.Equal(t, expected.String(), actual.String())
			},
		},
		{
			"Keep-3xx",
			[]resp.Fn{
				resp.Url(otherURL),
				resp.Code(http.StatusPermanentRedirect),
			},
			func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusPermanentRedirect, w.Code)

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)

				require.Equal(t, otherURL, actual.String())
			},
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		w := httptest.NewRecorder()
		d := resp.NewResponder()
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, w, r, d.Redirect(w, r, tc.fns...))
		})
	}
}

func TestResponderHtml(t *testing.T) {
	brokenTmpl := []byte("{{ define }}")
	tcs := []struct {
		name   string
		d      *resp.Responder
		fns    []resp.Fn
		assert testFn
	}{
		{
			name: "Zero-Value",
			d:    resp.NewResponder(),
			fns:  []resp.Fn{},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrBadConfig)
			},
		},
		{
			name: "With-Authed-Bad-Config",
			d:    resp.NewResponder(),
			fns:  []resp.Fn{resp.Authed()},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrBadConfig)
			},
		},
		{
			name: "With-Parser-Bad-Config",
			d:    resp.NewResponder(resp.WithParser(tt.NewParser())),
			fns:  []resp.Fn{},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrBadConfig)
			},
		},
		{
			name: "With-Parser-Tmpls",
			d:    resp.NewResponder(resp.WithParser(tt.NewParser(tt.NewMockFile("test.tmpl", nil)))),
			fns:  []resp.Fn{resp.Tmpls("test.tmpl")},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "Bad-Template-Syntax",
			d: resp.NewResponder(
				resp.WithParser(tt.NewParser(tt.NewMockFile("test.tmpl", brokenTmpl))),
			),
			fns: []resp.Fn{resp.Tmpls("test.tmpl")},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrBadConfig)

			},
		},
		{
			name: "With-Err-Tmpl-Bad-Syntax",
			d: resp.NewResponder(
				resp.WithParser(tt.NewParser(tt.NewMockFile("test.tmpl", brokenTmpl))),
				resp.WithErrTemplate("test.tmpl"),
			),
			fns: make([]resp.Fn, 0),
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.NotNil(t, err)
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "With-Err-Tmpl",
			d: resp.NewResponder(
				resp.WithParser(tt.NewParser(tt.NewMockFile("test.tmpl", nil))),
				resp.WithErrTemplate("test.tmpl"),
			),
			fns: make([]resp.Fn, 0),
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "With-Authed",
			d: resp.NewResponder(
				resp.WithParser(tt.NewParser(
					tt.NewMockFile("auth.tmpl", nil),
					tt.NewMockFile("test.tmpl", nil),
				)),
				resp.WithAuthTemplate("auth.tmpl"),
			),
			fns: []resp.Fn{resp.Authed(), resp.Tmpls("test.tmpl")},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
	}

	for _, tc := range tcs {
		// Arrange
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		s, err := session.NewStub(false).GetSession(r)
		require.Nil(t, err)

		ctx := context.WithValue(r.Context(), trails.SessionKey, s)
		ctx = context.WithValue(ctx, trails.CurrentUserKey, "I am definitely a user")
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()

		t.Run(tc.name, func(t *testing.T) {
			// Act + Assert
			tc.assert(t, w, r, tc.d.Html(w, r, tc.fns...))
		})
	}

	// NOTE(dlk): some sleight-of-hand here -
	// resp.Authed() is not used since it will error first before
	// reaching the checks in *Responder.Html;
	// this test verifies someone setting the Authed template
	// via another path (i.e., Tmpls) is gracefully handled.
	t.Run("With-Authed-No-User", func(t *testing.T) {
		// Arrange
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := context.WithValue(r.Context(), trails.SessionKey, session.Session{})
		ctx = context.WithValue(ctx, trails.CurrentUserKey, "I am definitely a user")
		w := httptest.NewRecorder()
		r = r.WithContext(ctx)

		responder := resp.NewResponder(
			resp.WithParser(tt.NewParser(tt.NewMockFile("err.tmpl", nil))),
			resp.WithAuthTemplate("auth.tmpl"),
			resp.WithErrTemplate("err.tmpl"),
		)

		// Act
		err := responder.Html(w, r, resp.Tmpls("auth.tmpl"))

		// Assert
		require.Nil(t, err)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestResponderSession(t *testing.T) {
	tcs := []struct {
		name        string
		ctx         context.Context
		expectedVal any
		expectedErr error
	}{
		{"Not-Set", context.Background(), session.Session{}, resp.ErrNotFound},
		{
			"Wrong-Key",
			context.WithValue(context.Background(), trails.Key("not-session-key"), session.Session{}),
			session.Session{},
			resp.ErrNotFound,
		},
		{
			"Set-With-Nil",
			context.WithValue(context.Background(), trails.SessionKey, nil),
			session.Session{},
			resp.ErrNotFound,
		},
		{
			"Set-With-Wrong-Type",

			context.WithValue(context.Background(), trails.SessionKey, struct{}{}),
			session.Session{},
			resp.ErrInvalid,
		},
		{
			"Set-With-session.Session",
			context.WithValue(context.Background(), trails.SessionKey, session.Session{}),
			session.Session{},
			nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := resp.NewResponder()
			actual, err := d.Session(tc.ctx)
			require.ErrorIs(t, err, tc.expectedErr)
			require.Equal(t, tc.expectedVal, actual)
		})
	}
}

func BenchmarkResponderRedirect(b *testing.B) {
	bcs := []struct {
		name string
		fns  []resp.Fn
	}{
		{"None", []resp.Fn{}},
		{"With-Code", []resp.Fn{resp.Code(http.StatusFound)}},
		{"With-Code-Overwrite", []resp.Fn{resp.Code(http.StatusTeapot)}},
		{"With-Param", []resp.Fn{resp.Params(map[string]string{"test": "true"})}},
		{"Url-Params", []resp.Fn{
			resp.Url("http://example.com/redirect"),
			resp.Params(map[string]string{
				"test":   "true",
				"go":     "fun",
				"params": "4",
				"good":   "times",
			}),
		}},
		{"4x-Params-Url-Redo", []resp.Fn{
			resp.Params(map[string]string{"test": "true"}),
			resp.Params(map[string]string{"go": "fun"}),
			resp.Params(map[string]string{"params": "4"}),
			resp.Params(map[string]string{"good": "times"}),
			resp.Url("http://example.com/redirect"),
		}},
	}

	for _, bc := range bcs {
		b.Run(bc.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
				w := httptest.NewRecorder()
				d := resp.NewResponder()
				d.Redirect(w, r, bc.fns...)
			}
		})
	}
}

func BenchmarkResponderJson(b *testing.B) {
	bcs := []struct {
		name string
		fns  []resp.Fn
	}{
		{"None", []resp.Fn{}},
		{"Code", []resp.Fn{resp.Code(200)}},
		{"Code-Data", []resp.Fn{resp.Code(200), resp.Data(map[string]string{"bench": "marks!"})}},
	}

	for _, bc := range bcs {
		b.Run(bc.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
				w := httptest.NewRecorder()
				d := resp.NewResponder()
				d.Json(w, r, bc.fns...)
			}
		})
	}
}

/*
func BenchmarkResponderRaw(b *testing.B) {
	bcs := [][]resp.Fn{
		{resp.Code(200)},
	}

	for _, bc := range bcs {
		for n := 0; n < b.N; n++ {
			r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			w := httptest.NewRecorder()
			d := resp.NewResponder()
			d.Raw(w, r, bc...)
		}
	}
}
*/

type testLogger struct {
	b *bytes.Buffer
}

func newLogger() testLogger                                  { return testLogger{bytes.NewBuffer(nil)} }
func (tl testLogger) AddSkip(i int) logger.Logger            { return tl }
func (tl testLogger) Skip() int                              { return 0 }
func (tl testLogger) Debug(msg string, _ *logger.LogContext) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Error(msg string, _ *logger.LogContext) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Info(msg string, _ *logger.LogContext)  { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Warn(msg string, _ *logger.LogContext)  { fmt.Fprint(tl.b, msg) }
