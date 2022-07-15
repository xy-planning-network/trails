package ranger

import "errors"

var (
	ErrBadConfig = errors.New("bad config")
	ErrNotExist  = errors.New("not exist")
	ErrNotValid  = errors.New("invalid")
)
