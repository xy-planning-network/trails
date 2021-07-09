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
	"github.com/xy-planning-network/trails/http/session"
)

func TestAuthed(t *testing.T) {
	key := "test"
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
			if tc.d.userSessionKey != "" {
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
		data map[string]interface{}
	}{
		{"Zero-Value", make(map[string]interface{})},
		{"Data", map[string]interface{}{"go": "rocks"}},
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
			if tc.data == nil {
				require.Equal(t, make(map[string]interface{}), r.data)
			} else {
				require.Equal(t, tc.data, r.data)
			}
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
			d := Responder{Logger: l}
			r := &Response{}

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
	key := "test"
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
			f:    session.Flash{Class: "success", Msg: "well done!"},
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
			NewResponder(WithLogger(newLogger()), WithSessionKey("key")),
			nil,
			func(t *testing.T, l testLogger, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Nil(t, l.Bytes())
				require.Equal(t, session.Flash{Class: "error", Msg: session.DefaultErrMsg}, s.Flashes(nil, nil)[0])
			},
		},
		{
			"With-Err-With-ContactUsErr",
			NewResponder(WithLogger(newLogger()), WithSessionKey("key"), WithContactErrMsg("howdy!")),
			ErrNotFound,
			func(t *testing.T, l testLogger, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Equal(t, ErrNotFound.Error(), l.String())
				require.Equal(t, session.Flash{Class: "error", Msg: "howdy!"}, s.Flashes(nil, nil)[0])
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			s := new(testFlashSession)
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			if tc.d.sessionKey != "" {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.sessionKey, s))
			}
			r := &Response{r: req}

			// Act
			err := GenericErr(tc.err)(*tc.d, r)

			// Assert
			tc.assert(t, tc.d.Logger.(testLogger), s, err)
		})
	}
}

func TestParam(t *testing.T) {
	goodURL, _ := url.Parse("http://example.com")

	testKey, testValue := "test", "params"
	withParams, _ := url.Parse("http://example.com")
	q := make(url.Values)
	q.Add(testKey, testValue)
	withParams.RawQuery = q.Encode()

	tcs := []struct {
		name   string
		r      *Response
		input  [2]string
		assert func(*testing.T, *Response, error)
	}{
		{
			name:  "No-Url",
			r:     &Response{},
			input: [2]string{"go", "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrMissingData)
			},
		},
		{
			name:  "Url",
			r:     &Response{url: goodURL},
			input: [2]string{"go", "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				params := r.url.Query()
				require.Equal(t, "rocks", params.Get("go"))
			},
		},
		{
			name:  "With-Params",
			r:     &Response{url: withParams},
			input: [2]string{"go", "rocks"},
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
			err := Param(tc.input[0], tc.input[1])(d, tc.r)

			// Assert
			tc.assert(t, tc.r, err)
		})
	}

	t.Run("Multiple", func(t *testing.T) {
		// Arrange
		r := &Response{url: goodURL}
		d := Responder{}
		ins := [][2]string{{"go", "rocks"}, {"fun", "tests"}}
		for _, in := range ins {
			// Act
			err := Param(in[0], in[1])(d, r)

			// Assert
			require.Nil(t, err)
		}

		require.Equal(t, "rocks", r.url.Query().Get("go"))
		require.Equal(t, "tests", r.url.Query().Get("fun"))
	})
}

func TestProps(t *testing.T) {
	ctxKey := "ctx"
	userKey := "user"
	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		props  map[string]interface{}
		assert func(*testing.T, *Response, error)
	}{
		{
			name:  "Zero-Value",
			d:     Responder{},
			r:     &Response{},
			props: nil,
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrNoUser)
			},
		},
		{
			name:  "No-CurrentUser",
			d:     Responder{},
			r:     &Response{},
			props: map[string]interface{}{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.ErrorIs(t, err, ErrNoUser)
				_, ok := r.data["initialProps"]
				require.False(t, ok)
			},
		},
		{
			name:  "With-CurrentUser",
			d:     Responder{userSessionKey: userKey},
			r:     &Response{},
			props: map[string]interface{}{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				i, ok := r.data["initialProps"]
				require.True(t, ok)

				p, ok := i.(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "test", p["currentUser"])

				require.Equal(t, "rocks", r.data["go"])
			},
		},
		{
			name:  "No-CurrentUser-With-User",
			d:     Responder{},
			r:     &Response{user: "test"},
			props: map[string]interface{}{"go": "rocks"},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				i, ok := r.data["initialProps"]
				require.True(t, ok)

				p, ok := i.(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "test", p["currentUser"])

				require.Equal(t, "rocks", r.data["go"])
			},
		},
		{
			name:  "Nil-Map",
			d:     Responder{},
			r:     &Response{user: "test"},
			props: nil,
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				i, ok := r.data["initialProps"]
				require.True(t, ok)

				p, ok := i.(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "test", p["currentUser"])
			},
		},
		{
			name:  "With-CtxKeys",
			d:     Responder{ctxKeys: []string{ctxKey}},
			r:     &Response{user: "test"},
			props: make(map[string]interface{}),
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)

				i, ok := r.data["initialProps"]
				require.True(t, ok)

				p, ok := i.(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, 1, p[ctxKey])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			req = req.WithContext(context.WithValue(req.Context(), ctxKey, 1))
			if tc.d.userSessionKey != "" {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.userSessionKey, "test"))
			}
			tc.r.r = req

			// Act
			err := Props(tc.props)(tc.d, tc.r)

			// Assert
			tc.assert(t, tc.r, err)
		})
	}
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
			NewResponder(WithSessionKey("key")),
			func(t *testing.T, code int, s session.FlashSessionable, err error) {
				require.Nil(t, err)
				require.Equal(t, http.StatusOK, code)
				require.Equal(t, session.Flash{Class: "success", Msg: "success!"}, s.Flashes(nil, nil)[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			s := new(testFlashSession)
			if tc.d.sessionKey != "" {
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

func TestUser(t *testing.T) {
	tcs := []struct {
		name string
		user interface{}
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
			err := User(tc.user)(d, r)

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
		err := User(struct{}{})(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, struct{}{}, r.user)

		// Arrange + Act
		err = User(1)(d, r)

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
	tcs := []struct {
		name   string
		d      *Responder
		r      *Response
		entry  string
		assert func(*testing.T, []string, map[string]interface{}, error)
	}{
		{
			"Zero-Value",
			NewResponder(),
			&Response{},
			"",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Nil(t, tmpls)
				require.Nil(t, data)
			},
		},
		{
			"With-Tmpls",
			NewResponder(),
			&Response{tmpls: []string{"test.tmpl"}},
			"",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Len(t, tmpls, 1)
				require.Nil(t, data)
			},
		},
		{
			"With-Data",
			NewResponder(),
			&Response{data: map[string]interface{}{"test": "data"}},
			"",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Nil(t, tmpls)
				require.Equal(t, "data", data["test"])
			},
		},
		{
			"With-Vue-No-Entry",
			NewResponder(WithVueTemplate("vue.tmpl")),
			&Response{},
			"",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Nil(t, tmpls)
				require.Nil(t, data)
			},
		},
		{
			"With-Vue",
			NewResponder(WithVueTemplate("vue.tmpl")),
			&Response{},
			"test",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[0])
				require.Equal(t, "test", data["entry"])
			},
		},
		{
			"With-Vue-With-Tmpls",
			NewResponder(WithVueTemplate("vue.tmpl")),
			&Response{tmpls: []string{"test.tmpl"}},
			"test",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[1])
				require.Equal(t, "test", data["entry"])
			},
		},
		{
			"With-Vue-With-Tmpls-With-Data",
			NewResponder(WithVueTemplate("vue.tmpl")),
			&Response{tmpls: []string{"test.tmpl"}, data: map[string]interface{}{"entry": "not-test", "other": 1}},
			"test",
			func(t *testing.T, tmpls []string, data map[string]interface{}, err error) {
				require.Nil(t, err)
				require.Equal(t, "vue.tmpl", tmpls[1])
				require.Equal(t, "test", data["entry"])
				require.Equal(t, 1, data["other"])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := Vue(tc.entry)(*tc.d, tc.r)

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
			NewResponder(WithLogger(newLogger()), WithSessionKey("key")),
			"Hey! Listen!",
			func(t *testing.T, expected string, s session.FlashSessionable, l testLogger, err error) {
				require.Nil(t, err)
				require.Equal(t, expected, l.String())
				require.Equal(t, session.Flash{Class: "warning", Msg: expected}, s.Flashes(nil, nil)[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			s := new(testFlashSession)
			if tc.d.sessionKey != "" {
				req = req.WithContext(context.WithValue(req.Context(), tc.d.sessionKey, s))
			}
			r := &Response{r: req}

			// Act
			err := Warn(tc.msg)(*tc.d, r)

			// Assert
			l, ok := tc.d.Logger.(testLogger)
			require.True(t, ok)
			tc.assert(t, tc.msg, s, l, err)
		})
	}
}

type testLogger struct {
	*bytes.Buffer
}

func newLogger() testLogger { return testLogger{new(bytes.Buffer)} }

func (tl testLogger) Debug(msg string, _ map[string]interface{}) { fmt.Fprint(tl, msg) }
func (tl testLogger) Error(msg string, _ map[string]interface{}) { fmt.Fprint(tl, msg) }
func (tl testLogger) Fatal(msg string, _ map[string]interface{}) { fmt.Fprint(tl, msg) }
func (tl testLogger) Info(msg string, _ map[string]interface{})  { fmt.Fprint(tl, msg) }
func (tl testLogger) Warn(msg string, _ map[string]interface{})  { fmt.Fprint(tl, msg) }

type testFlashSession []session.Flash

func (tfs testFlashSession) Flashes(_ http.ResponseWriter, _ *http.Request) []session.Flash {
	return tfs
}

func (tfs *testFlashSession) SetFlash(_ http.ResponseWriter, _ *http.Request, f session.Flash) error {
	*tfs = testFlashSession([]session.Flash{f})
	return nil
}
