package ranger

import "errors"

var (
	ErrBadConfig = errors.New("bad config")
	ErrNotValid  = errors.New("invalid")
)
