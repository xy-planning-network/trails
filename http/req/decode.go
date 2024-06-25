package req

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gorilla/schema"
	"github.com/xy-planning-network/trails"
)

func newQueryParamDecoder() *schema.Decoder {
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)

	return dec
}

// translateDecoderError converts an error returned by *schema.Decoder into standardized errors.
// Some *schema.Decoder errors are issues with calling code;
// some errors are unexpected issues;
// still some are issues with mismatches between a request's query params and the expected shape.
func translateDecoderError(err error) error {
	var pkgErrs schema.MultiError
	// NOTE(dlk): In testing the schema package, outside other errors handled above,
	// the package appears to always use MultiError to wrap errors up.
	// This is the "happy path".
	if !errors.As(err, &pkgErrs) {
		// TODO(dlk): Calling everything we aren't handling an ErrBadFormat could be misleading,
		// but wait and see as these pop up in the wild to figure out how to translate.
		return fmt.Errorf("%w: %s", trails.ErrBadFormat, err)
	}

	var validErrs ValidationErrors
	for _, pkgErr := range pkgErrs {
		switch err := pkgErr.(type) {
		case schema.ConversionError:
			ve := ValidationError{
				Field: err.Key,
				// NOTE(dlk): For non-slice values, ce.Index is -1.
				// Having such a subtle difference is confusing.
				Got:  fmt.Sprintf("bad value at index %d", max(0, err.Index)),
				Rule: fmt.Sprintf("must be " + err.Type.String()),
			}

			validErrs = append(validErrs, ve)

		case schema.EmptyFieldError:
			return fmt.Errorf(`%w: use validate pkg to set "required" fields, not schema`, trails.ErrNotImplemented)

		case schema.UnknownKeyError:
			// NOTE(dlk): We are currently accepting unknown keys,
			// as set in the default configuration for schema.Decoder.
			// That configuration can change.
			// We should gracefully handle that situation changing.
			ve := ValidationError{
				Field: err.Key,
				Got:   fmt.Sprint("value is set"),
				Rule:  fmt.Sprint("unexpected key should not be set"),
			}

			validErrs = append(validErrs, ve)

		default:
			// NOTE(dlk): This is an unfortuntate footgun with struct tags.
			// A field that requires, but that does not have a schema.Converter registered,
			// will not raise an error until a url.Values has the key set for the incorrectly configured field.
			// For example, if field "a" requires a converter,
			// until a url.Values sets a value for "a", no error returns.
			if strings.Contains(err.Error(), "schema: converter not found for") {
				return fmt.Errorf("%w: cannot convert values into unsupported type", trails.ErrNotImplemented)
			}

			// NOTE(dlk): The above covers all the known struct-back errors schema returns.
			// If it isn't one of those, it's likely a programming error, and thus a show-stopper.
			// Let's surface these immediately.
			return fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		}
	}

	return validErrs
}
