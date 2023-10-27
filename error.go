package trails

import "errors"

var (
	ErrBadConfig      = errors.New("bad config")
	ErrMissingData    = errors.New("missing data")
	ErrNotExist       = errors.New("not exist")
	ErrNotFound       = errors.New("not found")
	ErrNotImplemented = errors.New("not implemented")
	ErrNotValid       = errors.New("invalid")
	ErrUnexpected     = errors.New("unexpected")
)
