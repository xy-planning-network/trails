package session

import "errors"

var (
	ErrNotValid = errors.New("not valid")
	ErrNoUser   = errors.New("no user")
)
