package req

import (
	"reflect"
	"strings"

	v10 "github.com/go-playground/validator/v10"
	"github.com/xy-planning-network/trails"
)

// newValidator constructs a validator, which applies default configuration.
func newValidator() *v10.Validate {
	v := v10.New()
	v.RegisterValidation("enum", validateEnumerable)
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			name = ""
		}

		if name == "" {
			name = strings.SplitN(field.Tag.Get("schema"), ",", 2)[0]
		}

		if name == "-" {
			name = ""
		}

		return name
	})

	return v
}

// translateValidationErrors converts each issue into a ValidationError,
// returning them all as ValidationErrors.
func translateValidationErrors(errs v10.ValidationErrors) error {
	var validateErrs ValidationErrors
	for _, ve := range errs {
		field := ve.Namespace()

		ns := strings.SplitN(field, ".", 2)
		if len(ns) == 2 {
			field = ns[1]
		}

		rule := ve.Tag()
		if ve.Param() != "" {
			rule += "=" + ve.Param()
		}
		rule += "; " + ve.Type().String()

		validateErrs = append(validateErrs, ValidationError{
			Field: field,
			Got:   ve.Value(),
			Rule:  rule,
		})
	}

	return validateErrs
}

// validateEnumerable validates whether field is a valid Enumerable or slice of valid Enumerable.
func validateEnumerable(fl v10.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.Slice {
		vals := []reflect.Value{}
		for i := 0; i < field.Len(); i++ {
			vals = append(vals, field.Index(i))
		}

		return checkEnums(vals...)
	}

	return checkEnums(field)
}

// checkEnums asserts each [reflect.Value] is an Enumerable and valid.
func checkEnums(items ...reflect.Value) bool {
	if len(items) == 0 {
		return false
	}

	for _, item := range items {
		enum, ok := item.Interface().(trails.Enumerable)
		if err := enum.Valid(); !ok || err != nil {
			return false
		}
	}

	return true
}
