package resp

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseAuthed(t *testing.T) {
	key := "test"
	expected := &template.Template{}
	unauthed := &template.Template{}

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
				require.Nil(t, err)
				require.Nil(t, r.tmpls)
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
			name: "With-User",
			d:    Responder{userSessionKey: key},
			r:    &Response{},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Nil(t, r.tmpls)
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
			r:    &Response{tmpls: []*template.Template{expected}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 1)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "Tmpl-Unauthed",
			d:    Responder{authed: expected, userSessionKey: key, unauthed: unauthed},
			r:    &Response{tmpls: []*template.Template{unauthed}},
			assert: func(t *testing.T, r *Response, err error) {
				require.Nil(t, err)
				require.Len(t, r.tmpls, 1)
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "Tmpls",
			d:    Responder{authed: expected, userSessionKey: key},
			r:    &Response{tmpls: []*template.Template{{}, {}}},
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

func TestResponseCode(t *testing.T) {
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

func TestResponseData(t *testing.T) {
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

func TestResponseErr(t *testing.T) {
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
				require.Equal(t, tc.err.Error(), l.b.String())
			}
		})
	}

}

func TestResponseFlash(t *testing.T) {

}

func TestResponseGenericErr(t *testing.T) {

}

func TestResponseParam(t *testing.T) {
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

func TestResponseProps(t *testing.T) {
	tcs := []struct {
		name  string
		props map[string]interface{}
	}{
		{name: "Nil", props: nil},
		{name: "Zero-Value", props: make(map[string]interface{})},
		{name: "With-Props", props: map[string]interface{}{"go": "rocks"}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			d := Responder{}
			r := &Response{}

			// Act
			err := Props(tc.props)(d, r)

			// Assert
			require.Nil(t, err)
			require.Equal(t, tc.props, r.data["props"])
		})
	}

	t.Run("Repeat", func(t *testing.T) {
		// Arrange
		d := Responder{}
		r := &Response{}
		expected := map[string]interface{}{"go": "rocks"}

		// Act
		err := Props(expected)(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, expected, r.data["props"])

		// Arrange
		expected = map[string]interface{}{"now": "me"}

		// Act
		err = Props(expected)(d, r)

		// Assert
		require.Nil(t, err)
		require.Equal(t, expected, r.data["props"])
	})
}

func TestResponseSuccess(t *testing.T) {
	// TODO
}

func TestResponseTmpls(t *testing.T) {
	expected := &template.Template{}
	tcs := []struct {
		name  string
		tmpls []*template.Template
	}{
		{name: "Nil", tmpls: nil},
		{name: "Zero-Value", tmpls: []*template.Template{}},
		{name: "Tmpls", tmpls: []*template.Template{expected, expected}},
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

func TestResponseUnauthed(t *testing.T) {
	expected := &template.Template{}
	authed := &template.Template{}
	tcs := []struct {
		name   string
		d      Responder
		r      *Response
		assert func(*testing.T, *Response)
	}{
		{
			name: "Zero-Value",
			d:    Responder{},
			r:    &Response{},
			assert: func(t *testing.T, r *Response) {
				require.Nil(t, r.tmpls)
			},
		},
		{
			name: "With-Unauthed",
			d:    Responder{unauthed: expected},
			r:    &Response{},
			assert: func(t *testing.T, r *Response) {
				require.Equal(t, expected, r.tmpls[0])
			},
		},
		{
			name: "With-Unauthed-Repeat",
			d:    Responder{unauthed: expected},
			r:    &Response{tmpls: []*template.Template{expected}},
			assert: func(t *testing.T, r *Response) {
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 1)
			},
		},
		{
			name: "With-Authed",
			d:    Responder{authed: authed},
			r:    &Response{tmpls: []*template.Template{authed}},
			assert: func(t *testing.T, r *Response) {
				require.Equal(t, authed, r.tmpls[0])
				require.Len(t, r.tmpls, 1)
			},
		},
		{
			name: "With-Authed-With-Unauthed",
			d:    Responder{authed: authed, unauthed: expected},
			r:    &Response{tmpls: []*template.Template{authed}},
			assert: func(t *testing.T, r *Response) {
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 1)
			},
		},
		{
			name: "With-Tmpls",
			d:    Responder{unauthed: expected},
			r:    &Response{tmpls: []*template.Template{{}, {}}},
			assert: func(t *testing.T, r *Response) {
				require.Equal(t, expected, r.tmpls[0])
				require.Len(t, r.tmpls, 3)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange + Act
			err := Unauthed()(tc.d, tc.r)

			// Assert
			require.Nil(t, err)
			tc.assert(t, tc.r)
		})
	}
}

func TestResponseUser(t *testing.T) {
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

func TestResponseUrl(t *testing.T) {
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

func TestResponseWarn(t *testing.T) {

}

func TestResponseVue(t *testing.T) {

}

type testLogger struct {
	b *bytes.Buffer
}

func newLogger() testLogger                                      { return testLogger{bytes.NewBuffer(nil)} }
func (tl testLogger) Debug(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Error(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Fatal(msg string, _ map[string]interface{}) { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Info(msg string, _ map[string]interface{})  { fmt.Fprint(tl.b, msg) }
func (tl testLogger) Warn(msg string, _ map[string]interface{})  { fmt.Fprint(tl.b, msg) }
