package resp

import "errors"

var (
	ErrBadConfig   = errors.New("improperly configured")
	ErrDone        = errors.New("request ctx done")
	ErrInvalid     = errors.New("invalid")
	ErrMissingData = errors.New("missing data")
	ErrNotFound    = errors.New("not found")
	ErrNoUser      = errors.New("no user")
)
