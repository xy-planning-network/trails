package resp

import "errors"

var (
	ErrDone        = errors.New("request ctx done")
	ErrInvalid     = errors.New("invalid")
	ErrMissingData = errors.New("missing data")
	ErrNotFound    = errors.New("not found")
	ErrNoUser      = errors.New("no user")
	ErrRTFM        = errors.New("improperly called Do")
)
