package req_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/req"
)

type testEnum string

func (testEnum) String() string { return string("test") }
func (t testEnum) Valid() error {
	if t == "ignore" {
		return nil
	}
	return errors.New("oops")
}

func TestParserParseBody(t *testing.T) {
	// Arrange
	parser := req.NewParser()

	var actual req.ValidationErrors

	type test struct {
		A string `json:"a,omitempty" validate:"required"`
		B int64  `json:"b" validate:"gt=10,required"`
		C struct {
			Nested bool `json:"nested" validate:"eq=true"`
		} `json:"c"`
		D testEnum   `json:"d" validate:"enum"`
		E []testEnum `json:"e" validate:"enum"`
		F string     `json:"-"`
	}
	var input, output test

	b := new(bytes.Buffer)
	require.Nil(t, json.NewEncoder(b).Encode(input))

	// Act
	err := parser.ParseBody(b, struct{}{})

	// Assert
	require.ErrorIs(t, err, trails.ErrBadAny)

	// Arrange
	b.Reset()
	b.WriteByte('\x00')

	// Act
	err = parser.ParseBody(b, &output)

	// Assert
	require.ErrorIs(t, err, trails.ErrBadFormat)

	// Arrange
	expected := req.ValidationErrors{
		req.ValidationError{
			Field: "a",
			Got:   "",
			Rule:  "required; string",
		},
		req.ValidationError{
			Field: "b",
			Got:   int64(0),
			Rule:  "gt=10; int64",
		},
		req.ValidationError{
			Field: "c.nested",
			Got:   false,
			Rule:  "eq=true; bool",
		},
		req.ValidationError{
			Field: "d",
			Got:   testEnum(""),
			Rule:  "enum; req_test.testEnum",
		},
		req.ValidationError{
			Field: "e",
			Got:   []testEnum(nil),
			Rule:  "enum; []req_test.testEnum",
		},
	}

	require.Nil(t, json.NewEncoder(b).Encode(input))

	// Act
	err = parser.ParseBody(b, &output)

	// Assert
	require.ErrorIs(t, err, trails.ErrNotValid)
	require.Equal(t, input, output)
	require.ErrorAs(t, err, &actual)
	require.Len(t, actual, 5)
	require.Equal(t, expected[0], actual[0])
	require.Equal(t, expected[1], actual[1])
	require.Equal(t, expected[2], actual[2])
	require.Equal(t, expected[3], actual[3])
	require.Equal(t, expected[4], actual[4])

	// Arrange
	input.A = "hello"
	input.B = 20
	input.C.Nested = true
	input.D = "ignore"
	input.E = []testEnum{"ignore"}
	input.F = "ignore"

	b = new(bytes.Buffer)
	require.Nil(t, json.NewEncoder(b).Encode(input))

	// Act
	err = parser.ParseBody(b, &output)

	// Assert
	require.Nil(t, err)
	require.Equal(t, input.A, output.A)
	require.Equal(t, input.B, output.B)
	require.Equal(t, input.C, output.C)
	require.Equal(t, input.D, output.D)
	require.Equal(t, input.E, output.E)
	require.Equal(t, "", output.F)
}

func TestParserParseQueryParams(t *testing.T) {
	// Arrange
	parser := req.NewParser()
	u := make(url.Values)

	// Act
	err := parser.ParseQueryParams(u, struct{}{})

	// Assert
	require.ErrorIs(t, err, trails.ErrBadAny)

	// Act
	err = parser.ParseQueryParams(u, new(struct {
		A string `schema:"a,required"`
	}))

	// Assert
	require.ErrorIs(t, err, trails.ErrNotImplemented)

	// Arrange
	u.Set("a", "test")

	// Act
	err = parser.ParseQueryParams(u, new(struct {
		A struct{} `schema:"a"`
	}))

	// Assert
	require.ErrorIs(t, err, trails.ErrNotImplemented)

	// Arrange
	type test struct {
		A string   `schema:"a" validate:"required"`
		B int64    `schema:"b" validate:"gt=10,required"`
		C []string `schema:"c" validate:"len=2,required"`
		D string   `schema:"-"`
	}

	u.Set("b", "test")

	var actual req.ValidationErrors
	expected := req.ValidationErrors{{
		Field: "b",
		Got:   "bad value at index 0",
		Rule:  "must be int64",
	}}

	// Act
	err = parser.ParseQueryParams(u, new(test))

	// Assert
	require.ErrorIs(t, err, trails.ErrNotValid)
	require.ErrorAs(t, err, &actual)
	require.Len(t, expected, 1)
	require.Equal(t, expected[0], actual[0])

	// Arrange
	u.Set("b", "1")
	u.Add("c", "1")

	expected = req.ValidationErrors{
		{
			Field: "b",
			Got:   int64(1),
			Rule:  "gt=10; int64",
		},
		{
			Field: "c",
			Got:   []string{"1"},
			Rule:  "len=2; []string",
		},
	}

	// Act
	err = parser.ParseQueryParams(u, new(test))

	// Assert
	require.ErrorIs(t, err, trails.ErrNotValid)
	require.ErrorAs(t, err, &actual)
	require.Len(t, expected, 2)
	require.Equal(t, expected[0], actual[0])
	require.Equal(t, expected[1], actual[1])

	// Arrange
	u.Set("b", "20")
	u.Add("c", "2")
	u.Set("d", "ignore")
	actualVal := new(test)

	// Act
	err = parser.ParseQueryParams(u, actualVal)

	// Assert
	require.Nil(t, err)
	require.Equal(t, "test", actualVal.A)
	require.Equal(t, int64(20), actualVal.B)
	require.Equal(t, []string{"1", "2"}, actualVal.C)
	require.Equal(t, "", actualVal.D)
}
