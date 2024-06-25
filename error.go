package trails

import "errors"

var (
	ErrBadAny         = errors.New("bad value for any")
	ErrBadConfig      = errors.New("bad config")
	ErrBadFormat      = errors.New("bad format")
	ErrMissingData    = errors.New("missing data")
	ErrNotExist       = errors.New("not exist")
	ErrNotImplemented = errors.New("not implemented")
	ErrNotValid       = errors.New("invalid")
	ErrUnexpected     = errors.New("unexpected")
)
