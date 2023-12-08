package trails

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"
)

type Modelable interface {
	Exists() bool
}

// A Model is the essential data points for primary ID-based models in a trails application,
// indicating when a record was created, last updated and soft deleted.
type Model struct {
	ID        uint        `db:"id" json:"id"`
	CreatedAt time.Time   `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time   `db:"updated_at" json:"updatedAt"`
	DeletedAt DeletedTime `db:"deleted_at" json:"deletedAt"`
}

func (m Model) Exists() bool { return !m.CreatedAt.IsZero() }

// DeletedTime is a nullable timestamp marking a record as soft deleted.
type DeletedTime struct {
	sql.NullTime
}

// IsDeleted asserts whether the record is soft deleted.
func (dt DeletedTime) IsDeleted() bool { return !dt.Valid }

// AccessState is a string representation of the broadest, general access
// an entity such as an Account or a User has to a trails application.
type AccessState string

const (
	AccessGranted     AccessState = "granted"
	AccessInvited     AccessState = "invited"
	AccessRevoked     AccessState = "revoked"
	AccessVerifyEmail AccessState = "verify-email"
)

// String stringifies the AccessState.
//
// String implements fmt.Stringer.
func (as AccessState) String() string { return string(as) }

// CastAll translates source into []T.
// CastAll behaves similar to CastOne;
// refer to its documentation first for expected behavior,
// especially around error handling.
//
// CastAll differs from CastOne by expecting source to be a slice (or pointer to a slice).
// For each item in source, CastAll translates it to a T.
// Then, CastAll returns the collection of Ts as a []T.
func CastAll[T any](source any, orig error) (dest []T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Join(orig, fmt.Errorf("%w: panic: %s", ErrUnexpected, r))
		}
	}()

	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Pointer {
		sourceVal = sourceVal.Elem()
	}

	if sourceVal.Kind() != reflect.Slice {
		return nil, errors.Join(orig, fmt.Errorf("%w: source is not a slice", ErrNotImplemented))
	}

	var item T
	dest = make([]T, sourceVal.Len())
	switch any(item).(type) {
	case map[string]any:
		for i := 0; i < sourceVal.Len(); i++ {
			m, err := dumpToMap(sourceVal.Index(i))
			if err != nil {
				return dest, errors.Join(orig, err)
			}
			dest[i] = any(m).(T)
		}

	case Modelable:
		for i := 0; i < sourceVal.Len(); i++ {
			var item T
			itemVal := reflect.ValueOf(&item).Elem()
			if err := mapBetween(itemVal, sourceVal.Index(i)); err != nil {
				return dest, errors.Join(orig, err)
			}

			dest[i] = item
		}
	}

	return dest, orig
}

// CastOne translates source into a T or handles err.
// CastOne is a helper for converting the datatype provided by a database call
// into the desired target datatype T.
// CastOne requires source to be a struct
// and a "db" tag specifying the name of the database column be set on each field.
//
// CastOne attempts to translate source into one of the supported types.
// The main use case is where T implements Modelable.
// In that case, CastOne matches fields between T and source using the "db" tag.
// If T is a map[string]any, the keys in the map are the "db" tag for each field.
// If T is an unsupported type, CastOne returns ErrNotImplemented.
//
// With errors.Join, CastOne combines any errors it creates to orig.
// CastOne may return a valid T and error,
// enabling the caller to handle them as desired.
//
// Panics are recovered and returned as ErrUnexpected.
func CastOne[T any](source any, orig error) (dest T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Join(orig, fmt.Errorf("%w: panic: %s", ErrUnexpected, r))
		}
	}()

	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Pointer {
		sourceVal = sourceVal.Elem()
	}

	if sourceVal.Kind() != reflect.Struct {
		return dest, errors.Join(orig, fmt.Errorf("%w: source must be a struct or a pointer to one", ErrNotImplemented))
	}

	switch any(dest).(type) {
	case map[string]any:
		t, err := dumpToMap(sourceVal)
		if err != nil {
			return dest, errors.Join(orig, err)
		}

		dest, _ = any(t).(T)

	case Modelable:
		destVal := reflect.ValueOf(&dest).Elem()
		if err := mapBetween(destVal, sourceVal); err != nil {
			// TODO(dlk): mapBetween may set some fields on dest,
			// but still throw an error. Reset dest?
			return dest, errors.Join(orig, err)
		}

		dest, _ = destVal.Interface().(T)

	default:
		err = fmt.Errorf("%w: unhandled translate for %T", ErrNotImplemented, dest)
		return dest, errors.Join(orig, err)
	}

	return dest, orig
}

func dumpToMap(source reflect.Value) (map[string]any, error) {
	m := make(map[string]any)
	for _, sourceField := range reflect.VisibleFields(source.Type()) {
		tag := sourceField.Tag.Get("db")
		sourceVal := source.FieldByIndex(sourceField.Index)
		isIDField := tag == "id"
		noID := isIDField && sourceVal.IsZero()
		if isIDField && noID {
			return nil, fmt.Errorf("%s %w", source.Type().Name, ErrNotExist)
		}

		m[tag] = sourceVal.Interface()
	}

	return m, nil
}

func mapBetween(dest, source reflect.Value) error {
	if dest.Kind() != reflect.Struct {
		return fmt.Errorf("%w: T must be a struct", ErrNotImplemented)
	}

	for _, sourceField := range reflect.VisibleFields(source.Type()) {
		sourceTag, ok := sourceField.Tag.Lookup("db")
		if !ok {
			err := fmt.Errorf(
				"%w: source field %q has no db tag for %s",
				ErrNotValid,
				sourceField.Name,
				source.Type().Name,
			)
			return err
		}

		sourceVal := source.FieldByIndex(sourceField.Index)
		isIDField := sourceTag == "id"
		noID := isIDField && sourceVal.IsZero()
		if isIDField && noID {
			return fmt.Errorf("%s %w", source.Type().Name, ErrNotExist)
		}

		var foundSource bool
		for _, destField := range reflect.VisibleFields(dest.Type()) {
			destTag := destField.Tag.Get("db")
			if destTag == sourceTag {
				foundSource = true
				dest.FieldByIndex(destField.Index).Set(sourceVal)
			}
		}

		if !foundSource {
			err := fmt.Errorf("%w: source tag %q not found on dest for %s", ErrNotValid, sourceTag, dest.Type().Name)
			return err
		}
	}

	return nil
}
