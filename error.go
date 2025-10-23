package trails

import "errors"

var (
	// ErrBadConfig means some configuration is incorrect
	// and likely needs changes to environmental variables
	// or parameters passed to services.
	ErrBadConfig = errors.New("bad config")

	// ErrExists means some data conflicts with existing data
	// that cannot be resolved gracefully.
	ErrExists = errors.New("exists")

	// ErrMissingData means required data was not provided.
	ErrMissingData = errors.New("missing data")

	// ErrNotExist means the primary key used to find a record
	// did not match any record.
	ErrNotExist = errors.New("not exist")

	// ErrNotFound means filters or other parameters
	// used to find one or many records did not match any records.
	ErrNotFound = errors.New("not found")

	// ErrUnaddressable means some value ought to be a pointer
	// so that the value can be updated with new data,
	// but it is not a pointer.
	ErrUnaddressable = errors.New("unaddressable")

	// ErrNotValid means some data does not conform to the expected shape for its type
	// or what it's type requires.
	ErrNotValid = errors.New("invalid")

	// ErrUnexpected catches all errors not otherwise handled.
	//
	// If ErrUnexpected returns, consider breaking out the case into its own error.
	ErrUnexpected = errors.New("unexpected")
)
