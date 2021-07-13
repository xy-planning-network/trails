package session

import "errors"

var (
	ErrFailedConfig = errors.New("failed config")
	ErrNotValid     = errors.New("not valid")
	ErrNoUser       = errors.New("no user")
)
