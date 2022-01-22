package resp

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/ctx"
	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/session"
	"github.com/xy-planning-network/trails/logger"
)

func TestAuthed(t *testing.T) {
	key := ctxKey("test")
	expected := "authed.tmpl"
	unauthed := "unauthed.tmpl"

	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		assert func(*testing.T, *Response, error)
	}{
		{
			name: "Zero-Value",
			d:    Responder{},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrBadConfig)
				require.Len(t, r.tmpls, 0)
			},
		},
		{
			name: "With-Auth",
			d:    Responder{authed: expected},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrNoUser)
				require.Len(t, r.tmpls, 0)
			},
		},
		{
			name: "With-User-With-Auth",
			d:    Responder{authed: expected, userSessionKey: key},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 1)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "Tmpl-Authed",
			d:    Responder{authed: expected, userSessionKey: key},
			r:    &Response{tmpls: []string{expected}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 1)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "Tmpl-Unauthed",
			d:    Responder{authed: expected, userSessionKey: key, unauthed: unauthed},
			r:    &Response{tmpls: []string{unauthed}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 1)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "Tmpls",
			d:    Responder{authed: expected, userSessionKey: key},
			r:    &Response{user: struct{}{}, tmpls: []string{"test.tmpl", "example.tmpl"}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 3)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			if tc.d.userSessionKey != nil {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.userSessionKey, 1))
			}
			tc.r.r = req

			// Act
			err := Authed()(tc.d, tc.r)

			// Assert
			tc.assert(t, tc.r, err)
		})
	}
}

func TestCode(t *testing.T) {
	tcs := []struct {
		name string
		code int
	}{
		{"Min-Int32", math.MinInt32},
		{"200", http.StatusOK},
		{"Max-Int32", math.MaxInt32},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := Code(tc.code)(d, r)

			// Assert
			require.Nil(t, err)
			require.Equal(t, tc.code, r.code)
		})
	}
}

func TestData(t *testing.T) {
	tcs := []struct {
		name string
		data map[string]any
	}{
		{"Zero-Value", make(map[string]any)},
		{"Data", map[string]any{"go": "rocks"}},
		{"Nil", nil},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := Data(tc.data)(d, r)

			// Assert
			require.Nil(t, err)
			require.Equal(t, tc.data, r.data)
		})
	}
}

func TestErr(t *testing.T) {
	tcs := []struct {
		name string
		err  error
	}{
		{name: "Zero-Value", err: nil},
		{name: "Error", err: ErrInvalid},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			l := newLogger()
			d := Responder{logger: l}
			r := &Response{r: httptest.NewRequest(http.MethodGet, "http://example.com", nil)}

			// Act
			err := Err(tc.err)(d, r)

			// Assert
			require.Nil(t, err)
			require.Equal(t, http.StatusInternalServerError, r.code)
			if tc.err != nil {
				require.Equal(t, tc.err.Error(), l.String())
			}
		})
	}

}

func TestFlash(t *testing.T) {
	key := ctxKey("test")
	tcs := []struct {
		name   string
		d      *Responder
		f      session.Flash
		assert func(*testing.T, session.FlashSessionable, session.Flash, error)
	}{
		{
			name: "No-Key",
			d:    NewResponder(),
			f:    session.Flash{},
			assert: func(t *testing.T, s session.FlashSessionable, _ session.Flash, err error) {
				require.ErrorIs(t, err, ErrNotFound)
				require.Nil(t, s.Flashes(nil, nil))
			},
		},
		{
			name: "With-Key",
			d:    NewResponder(WithSessionKey(key)),
			f:    session.Flash{Type: "success", Msg: "well done!"},
			assert: func(t *testing.T, s session.FlashSessionable, f session.Flash, err error) {
				require.Nil(t, err)
				require.Equal(t, f, s.Flashes(nil, nil)[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			s := new(testFlashSession)
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			ctx := context.WithValue(req.Context(), key, s)
			r := &Response{r: req.WithContext(ctx), w: w}

			// Act + Assert
			tc.assert(t, s, tc.f, Flash(tc.f)(*tc.d, r))
		})
	}
}

func TestGenericErr(t *testing.T) {
	tcs := []struct {
		name   string
		d      *Responder
		err    error
		assert func(*testing.T, testLogger, session.FlashSessionable, error)
	}{
		{
			"No-Session",
			NewResponder(WithLogger(newLogger())),
			nil,
			func(t *testing.T, l testLogger, s session.FlashSessionable, err error) {
				require.NotNil(t, err)
				require.Nil(t, l.Bytes())
				require.Nil(t, s.Flashes(nil, nil))
			},
		},
		{
			"With-Session-Nil-Err-DefaultErrMsg",
			NewResponder(WithLogger(newLogger()), WithSessionKey(ctxKey("key"))),
			nil,
			func(t *testing.T, l testLogger, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Nil(t, l.Bytes())
				require.Equal(t, session.Flash{Type: "error", Msg: session.DefaultErrMsg}, s.Flashes(nil, nil)[0])
			},
		},
		{
			"With-Err-With-ContactUsErr",
			NewResponder(WithLogger(newLogger()), WithSessionKey(ctxKey("key")), WithContactErrMsg("howdy!")),
			ErrNotFound,
			func(t *testing.T, l testLogger, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Equal(t, ErrNotFound.Error(), l.String())
				require.Equal(t, session.Flash{Type: "error", Msg: "howdy!"}, s.Flashes(nil, nil)[0])
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			s := new(testFlashSession)
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			if tc.d.sessionKey != nil {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.sessionKey, s))
			}
			r := &Response{r: req}

			// Act
			err := GenericErr(tc.err)(*tc.d, r)

			// Assert
			tc.assert(t, tc.d.logger.(testLogger), s, err)
		})
	}
}

func TestParams(t *testing.T) {
	goodURL, _ := url.Parse("http://example.com")

	testKey, testValue := "test", "params"
	withParams, _ := url.Parse("http://example.com")
	q := make(url.Values)
	q.Add(testKey, testValue)
	withParams.RawQuery = q.Encode()

	tcs := []struct {
		name   string
		r      *Response
		input  map[string]string
		assert func(*testing.T, *Response, error)
	}{
		{
			name:  "No-Url",
			r:     &Response{},
			input: map[string]string{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrMissingData)
			},
		},
		{
			name:  "Url",
			r:     &Response{url: goodURL},
			input: map[string]string{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				params := r.url.Query()
				require.Equal(t, "rocks", params.Get("go"))
			},
		},
		{
			name:  "With-Params",
			r:     &Response{url: withParams},
			input: map[string]string{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Equal(t, "rocks", r.url.Query().Get("go"))
				require.Equal(t, testValue, r.url.Query().Get(testKey))
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}

			// Act
			err := Params(tc.input)(d, tc.r)

			// Assert
			tc.assert(t, tc.r, err)
		})
	}

	t.Run("Multiple", func(t *testing.T) {
		// Arrange
		r := &Response{url: goodURL}
		d := Responder{}
		ins := map[string]string{"go": "rocks", "fun": "tests"}

		// Act
		err := Params(ins)(d, r)

		// Assert
		require.Nil(t, err)

		require.Equal(t, "rocks", r.url.Query().Get("go"))
		require.Equal(t, "tests", r.url.Query().Get("fun"))
	})
}

func TestSuccess(t *testing.T) {
	tcs := []struct {
		name   string
		d      *Responder
		assert func(*testing.T, int, session.FlashSessionable, error)
	}{
		{
			"No-Session",
			NewResponder(),
			func(t *testing.T, code int, s session.FlashSessionable, err error) {
				require.NotNil(t, err)
				require.Equal(t, http.StatusOK, code)
				require.Nil(t, s.Flashes(nil, nil))
			},
		},
		{
			"With-Session",
			NewResponder(WithSessionKey(ctxKey("key"))),
			func(t *testing.T, code int, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, code)
				require.Equal(t, session.Flash{Type: "success", Msg: "success!"}, s.Flashes(nil, nil)[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			s := new(testFlashSession)
			if tc.d.sessionKey != nil {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.sessionKey, s))
			}
			r := &Response{r: req}

			// Act
			err := Success("success!")(*tc.d, r)

			// Assert
			tc.assert(t, r.code, s, err)
		})
	}
}

func TestTmpls(t *testing.T) {
	expected := "example.tmpl"
	tcs := []struct {
		name  string
		tmpls []string
	}{
		{name: "Nil", tmpls: nil},
		{name: "Zero-Value", tmpls: []string{}},
		{name: "Tmpls", tmpls: []string{expected, expected}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := Tmpls(tc.tmpls...)(d, r)

			// Assert
			require.Nil(t, err)
			if len(tc.tmpls) == 0 {
				require.Nil(t, r.tmpls)
			} else {
				require.Equal(t, tc.tmpls, r.tmpls)
			}
		})
	}

	t.Run("Repeat", func(t *testing.T) {
		// Arrange
		d := Responder{}
		r := &Response{}

		// Act
		err := Tmpls(expected)(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, expected, r.tmpls[0])

		// Act
		err = Tmpls(expected)(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, expected, r.tmpls[1])
	})
}

func TestToRoot(t *testing.T) {
	good, err := url.ParseRequestURI("https://example.com/test")
	require.Nil(t, err)

	other, err := url.ParseRequestURI("https://example.com/other")
	require.Nil(t, err)
	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		assert func(t *testing.T, url *url.URL, err error)
	}{
		{
			"Zero-Value",
			Responder{},
			&Response{},
			func(t *testing.T, url *url.URL, err error) {
				require.ErrorIs(t, err, ErrMissingData)
			},
		},
		{
			"With-RootUrl",
			Responder{rootUrl: good},
			&Response{},
			func(t *testing.T, url *url.URL, err error) {
				require.Nil(t, err)
				require.Equal(t, good, url)
			},
		},
		{
			"Overwrite-Url",
			Responder{rootUrl: good},
			&Response{url: other},
			func(t *testing.T, url *url.URL, err error) {
				require.Nil(t, err)
				require.Equal(t, good, url)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := ToRoot()(tc.d, tc.r)

			// Assert
			tc.assert(t, tc.r.url, err)
		})
	}
}

func TestUnauthed(t *testing.T) {
	expected := "unauthed.tmpl"
	authed := "authed.tmpl"
	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		assert func(*testing.T, *Response, error)
	}{
		{
			name: "Zero-Value",
			d:    Responder{},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrBadConfig)
			},
		},
		{
			name: "With-Unauthed",
			d:    Responder{unauthed: expected},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "With-Unauthed-Repeat",
			d:    Responder{unauthed: expected},
			r:    &Response{tmpls: []string{expected}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 1)
			},
		},
		{
			name: "With-Only-Authed",
			d:    Responder{authed: authed},
			r:    &Response{tmpls: []string{authed}},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrBadConfig)
			},
		},
		{
			name: "With-Authed-With-Unauthed",
			d:    Responder{authed: authed, unauthed: expected},
			r:    &Response{tmpls: []string{authed}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 1)
			},
		},
		{
			name: "With-Tmpls",
			d:    Responder{unauthed: expected},
			r:    &Response{tmpls: []string{"test.tmpl", "example.tmpl"}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 3)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, tc.r, Unauthed()(tc.d, tc.r))
		})
	}
}

func TestCurrentUser(t *testing.T) {
	tcs := []struct {
		name string
		user any
	}{
		{name: "Nil", user: nil},
		{name: "Struct", user: struct{}{}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := CurrentUser(tc.user)(d, r)

			// Assert
			require.Nil(t, err)
			require.Equal(t, tc.user, r.user)
		})
	}

	t.Run("Repeat", func(t *testing.T) {
		// Arrange
		d := Responder{}
		r := &Response{}

		// Act
		err := CurrentUser(struct{}{})(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, struct{}{}, r.user)

		// Arrange + Act
		err = CurrentUser(1)(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, 1, r.user)
	})
}

func TestUrl(t *testing.T) {
	tcs := []struct {
		name   string
		url    string
		assert require.ErrorAssertionFunc
	}{
		{name: "Zero-Value", url: "", assert: require.Error},
		{name: "NUL-Byte", url: "\x00", assert: require.Error},
		{name: "URL", url: "http://example.com", assert: require.NoError},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := Url(tc.url)(d, r)

			// Assert
			tc.assert(t, err)
		})
	}
}

func TestVue(t *testing.T) {
	good, err := url.ParseRequestURI("https://example.com/test")
	require.Nil(t, err)
	aKey := ctxKey("ctx")
	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		entry  string
		assert func(*testing.T, []string, any, error)
	}{
		{
			"Zero-Value",
			Responder{},
			&Response{},
			"",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Nil(t, tmpls)
				require.Nil(t, data)
			},
		},
		{
			"With-Tmpls",
			Responder{},
			&Response{tmpls: []string{"test.tmpl"}},
			"",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Len(t, tmpls, 1)
				require.Nil(t, data)
			},
		},
		{
			"With-Vue-No-Entry",
			Responder{vue: "vue.tmpl"},
			&Response{},
			"",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Nil(t, tmpls)
				require.Nil(t, data)
			},
		},
		{
			"With-Vue",
			Responder{vue: "vue.tmpl"},
			&Response{},
			"test",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[0])

				actualData, ok := data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, "test", actualData["entry"])
			},
		},
		{
			"With-Vue-With-Tmpls",
			Responder{vue: "vue.tmpl"},
			&Response{tmpls: []string{"test.tmpl"}},
			"test",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[1])

				actualData, ok := data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, "test", actualData["entry"])
			},
		},
		{
			"With-CtxKeys",
			Responder{vue: "vue.tmpl", ctxKeys: []keyring.Keyable{aKey}},
			&Response{user: "test"},
			"test",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[0])

				actualData, ok := data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, "test", actualData["entry"])

				actualProps, ok := actualData["props"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, 1, actualProps[aKey.Key()])
			},
		},
		{
			"With-All",
			Responder{vue: "vue.tmpl", rootUrl: good, ctxKeys: []ctx.CtxKeyable{aKey}},
			&Response{user: 1, tmpls: []string{"test.tmpl"}, data: map[string]any{"entry": "not-test", "other": 1}},
			"test",
			func(t *testing.T, tmpls []string, data any, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[1])

				actualData, ok := data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, "test", actualData["entry"])

				actualProps, ok := actualData["props"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, 1, actualProps["other"])

				actualInit, ok := actualProps["initialProps"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, 1, actualInit["currentUser"])
				require.Equal(t, "https://example.com/test", actualInit["baseURL"])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			require.Nil(t, err)
			tc.r.r = req.Clone(context.WithValue(req.Context(), aKey, 1))

			// Act
			err = Vue(tc.entry)(tc.d, tc.r)

			// Assert
			tc.assert(t, tc.r.tmpls, tc.r.data, err)
		})
	}
}

func TestWarn(t *testing.T) {
	tcs := []struct {
		name   string
		d      *Responder
		msg    string
		assert func(*testing.T, string, session.FlashSessionable, testLogger, error)
	}{
		{
			"No-Sess-No-Msg",
			NewResponder(WithLogger(newLogger())),
			"",
			func(t *testing.T, expected string, s session.FlashSessionable, l testLogger, err error) {
				require.ErrorIs(t, err, ErrNotFound)
				require.Equal(t, expected, l.String())
				require.Nil(t, s.Flashes(nil, nil))
			},
		},
		{
			"No-Sess-With-Msg",
			NewResponder(WithLogger(newLogger())),
			"Hey! Listen!",
			func(t *testing.T, expected string, s session.FlashSessionable, l testLogger, err error) {
				require.ErrorIs(t, err, ErrNotFound)
				require.Equal(t, expected, l.String())
				require.Nil(t, s.Flashes(nil, nil))
			},
		},
		{
			"With-Sess-With-Msg",
			NewResponder(WithLogger(newLogger()), WithSessionKey(ctxKey("key"))),
			"Hey! Listen!",
			func(t *testing.T, expected string, s session.FlashSessionable, l testLogger, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, l.String())
				require.Equal(t, session.Flash{Type: "warning", Msg: expected}, s.Flashes(nil, nil)[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			s := new(testFlashSession)
			if tc.d.sessionKey != nil {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.sessionKey, s))
			}
			r := &Response{r: req}

			// Act
			err := Warn(tc.msg)(*tc.d, r)

			// Assert
			l, ok := tc.d.logger.(testLogger)
			require.True(t, ok)
			tc.assert(t, tc.msg, s, l, err)
		})
	}
}

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

type testFlashSession []session.Flash

func (tfs testFlashSession) ClearFlashes(_ http.ResponseWriter, _ *http.Request) { tfs = nil }

func (tfs testFlashSession) Flashes(_ http.ResponseWriter, _ *http.Request) []session.Flash {
	return tfs
}

func (tfs *testFlashSession) SetFlash(_ http.ResponseWriter, _ *http.Request, f session.Flash) error {
	*tfs = testFlashSession([]session.Flash{f})
	return nil
}
