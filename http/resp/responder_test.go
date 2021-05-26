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
	"github.com/xy-planning-network/trails/http/resp"
	"github.com/xy-planning-network/trails/http/session"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
)

type testFn func(*testing.T, *httptest.ResponseRecorder, *http.Request, error)

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

	/*
		t.Run("Status-Already-Written", func(t *testing.T) {
			// Arrange
			r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

			w := httptest.NewRecorder()
			w.WriteHeader(http.StatusOK)

			d := resp.NewResponder()

			// Act
			err := d.Raw(w, r, resp.Code(http.StatusTeapot))

			// Assert
			require.Nil(t, err)

			actual := w.Result()
			require.Equal(t, http.StatusOK, actual.StatusCode)
		})
	*/
}

func TestResponderCurrentUser(t *testing.T) {
	key := "user"
	tcs := []struct {
		name        string
		ctx         context.Context
		expectedVal interface{}
		expectedErr error
	}{
		{"Not-Set", context.Background(), nil, resp.ErrNotFound},
		{"Set-With-Nil", context.WithValue(context.Background(), key, nil), nil, resp.ErrNotFound},
		{"Set-With-Val", context.WithValue(context.Background(), key, struct{}{}), struct{}{}, nil},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := resp.NewResponder(resp.WithUserSessionKey(key))
			actual, err := d.CurrentUser(tc.ctx)
			require.ErrorIs(t, err, tc.expectedErr)
			require.Equal(t, tc.expectedVal, actual)
		})
	}

	t.Run("No-Key", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), key, struct{}{})
		d := resp.NewResponder()
		actual, err := d.CurrentUser(ctx)
		require.ErrorIs(t, err, resp.ErrNotFound)
		require.Nil(t, actual)
	})
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
		r := httptest.NewRequest("GET", "http://example.com", nil)
		w := httptest.NewRecorder()
		l := newLogger()
		d := resp.NewResponder(resp.WithLogger(l))
		t.Run(tc.name, func(t *testing.T) {
			d.Err(w, r, tc.expected)
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
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				require.Equal(t, []byte("{}\n"), w.Body.Bytes())
			},
		},
		{
			name: "With-Code",
			fns:  []resp.Fn{resp.Code(http.StatusTeapot)},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTeapot, w.Code)
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				require.Equal(t, []byte("{}\n"), w.Body.Bytes())
			},
		},
		{
			name: "With-Data",
			fns:  []resp.Fn{resp.Data(map[string]interface{}{"go": "rocks"})},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var b bytes.Buffer
				err = json.NewEncoder(&b).Encode(map[string]map[string]string{"data": {"go": "rocks"}})
				require.Nil(t, err)
				require.Equal(t, b.Bytes(), w.Body.Bytes())
			},
		},
		{
			name: "With-User",
			fns:  []resp.Fn{resp.User(1)},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))

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
				resp.User(1),
				resp.Data(map[string]interface{}{"go": "rocks"}),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTeapot, w.Code)
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var b bytes.Buffer
				err = json.NewEncoder(&b).
					Encode(
						map[string]interface{}{
							"currentUser": 1,
							"data":        map[string]string{"go": "rocks"},
						},
					)
				require.Nil(t, err)
				require.Equal(t, b.Bytes(), w.Body.Bytes())
			},
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest("GET", "http://example.com", nil)
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
				resp.Param("test", "true"),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrMissingData)
			},
		},
		{
			name: "Params4x-Url-Redirect",
			fns: []resp.Fn{
				resp.Param("test", "true"),
				resp.Param("go", "fun"),
				resp.Param("params", "4"),
				resp.Param("good", "times"),
				resp.Url("http://example.com/redirect"),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusFound, w.Code)

				expected, err := url.ParseRequestURI("http://example.com/redirect")
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
				resp.Url("http://example.com"),
				resp.Code(http.StatusTeapot),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusSeeOther, w.Code)

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)

				expected, err := url.ParseRequestURI("http://example.com")
				require.Nil(t, err)
				require.Equal(t, expected.String(), actual.String())
			},
		},
		{
			name: "Overwrite-5xx",
			fns: []resp.Fn{
				resp.Url("http://example.com"),
				resp.Code(http.StatusInsufficientStorage),
			},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusTemporaryRedirect, w.Code)

				actual, err := url.ParseRequestURI(w.Header().Get("Location"))
				require.Nil(t, err)

				expected, err := url.ParseRequestURI("http://example.com")
				require.Nil(t, err)
				require.Equal(t, expected.String(), actual.String())
			},
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest("GET", "http://example.com", nil)
		w := httptest.NewRecorder()
		d := resp.NewResponder()
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, w, r, d.Redirect(w, r, tc.fns...))
		})
	}
}

func TestResponderRender(t *testing.T) {
	sessionKey := "test"
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
			name: "With-Parser",
			d:    resp.NewResponder(resp.WithParser(tt.NewParser())),
			fns:  []resp.Fn{},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.ErrorIs(t, err, resp.ErrMissingData)
			},
		},
		{
			name: "With-Parser-Tmpls",
			d:    resp.NewResponder(resp.WithParser(tt.NewParser(tt.NewMockTmpl("test.tmpl", nil)))),
			fns:  []resp.Fn{resp.Tmpls("test.tmpl")},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Contains(t, err.Error(), "can't retrieve session")
			},
		},
		{
			name: "With-Parser-Tmpls-Session",
			d: resp.NewResponder(
				resp.WithParser(tt.NewParser(tt.NewMockTmpl("test.tmpl", nil))),
				resp.WithSessionKey(sessionKey),
			),
			fns: []resp.Fn{resp.Tmpls("test.tmpl")},
			assert: func(t *testing.T, w *httptest.ResponseRecorder, r *http.Request, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest("GET", "http://example.com", nil)
		ctx := context.WithValue(r.Context(), sessionKey, session.Stub{})
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, w, r, tc.d.Render(w, r, tc.fns...))
		})
	}
}

func TestResponderSession(t *testing.T) {
	key := "session"
	tcs := []struct {
		name        string
		ctx         context.Context
		expectedVal interface{}
		expectedErr error
	}{
		{"Not-Set", context.Background(), nil, resp.ErrNotFound},
		{"Set-With-Nil", context.WithValue(context.Background(), key, nil), nil, resp.ErrNotFound},
		{"Set-With-Val", context.WithValue(context.Background(), key, session.Stub{}), session.Stub{}, nil},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := resp.NewResponder(resp.WithSessionKey(key))
			actual, err := d.Session(tc.ctx)
			require.ErrorIs(t, err, tc.expectedErr)
			require.Equal(t, tc.expectedVal, actual)
		})
	}

	t.Run("No-Key", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), key, struct{}{})
		d := resp.NewResponder()
		actual, err := d.Session(ctx)
		require.ErrorIs(t, err, resp.ErrNotFound)
		require.Nil(t, actual)
	})
}

func BenchmarkResponderJson(b *testing.B) {
	bcs := [][]resp.Fn{
		{resp.Code(200)},
	}

	for _, bc := range bcs {
		for n := 0; n < b.N; n++ {
			r := httptest.NewRequest("GET", "http://example.com", nil)
			w := httptest.NewRecorder()
			d := resp.NewResponder()
			d.Json(w, r, bc...)
		}
	}
}

/*
func BenchmarkResponderRaw(b *testing.B) {
	bcs := [][]resp.Fn{
		{resp.Code(200)},
	}

	for _, bc := range bcs {
		for n := 0; n < b.N; n++ {
			r := httptest.NewRequest("GET", "http://example.com", nil)
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

func newLogger() testLogger                                      { return testLogger{bytes.NewBuffer(nil)} }
func (tl testLogger) Debug(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Error(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Fatal(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Info(msg string, _ map[string]interface{})  { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Warn(msg string, _ map[string]interface{})  { fmt.Fprint(tl.b, msg) }
