package trails

import "errors"

var (
	ErrBadConfig   = errors.New("bad config")
	ErrMissingData = errors.New("missing data")
	ErrNotExist    = errors.New("not exist")
	ErrNotValid    = errors.New("invalid")
)
