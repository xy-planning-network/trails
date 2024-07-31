package req_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/req"
)

func TestValidationErrorsError(t *testing.T) {
	// Arrange
	var v req.ValidationErrors

	// Act
	actual := v.Error()

	// Assert
	require.Zero(t, actual)

	// Arrange
	v = append(
		v,
		req.ValidationError{
			Field: "first",
			Rule:  "required; string",
		},
		req.ValidationError{
			Field: "second",
			Got:   "big boo boo",
			Rule:  "len=1; string",
		},
	)

	expected := strings.Join([]string{
		`field="first" rule="required; string" got="<nil>"`,
		`field="second" rule="len=1; string" got="big boo boo"`,
	}, "\n")

	// Act
	actual = v.Error()

	// Assert
	require.Equal(t, expected, actual)
}

func TestValidationErrorsMarshalJSON(t *testing.T) {
	// Arrange
	var v req.ValidationErrors

	// Act
	actual, err := json.Marshal(v)

	// Assert
	require.Nil(t, err)
	require.Equal(t, "{}", string(actual))

	// Arrange
	v = append(v, req.ValidationError{
		Field: "first",
		Rule:  "required; string",
		Got:   "",
	})

	expected := `{"validationErrors":[{"field":"first","got":"","rule":"required; string"}]}`

	// Act
	actual, err = json.Marshal(v)

	// Assert
	require.Nil(t, err)
	require.Equal(t, expected, string(actual))
}

func TestValidationErrorsUnwrap(t *testing.T) {
	require.ErrorIs(t, req.ValidationErrors{}, trails.ErrNotValid)
}
