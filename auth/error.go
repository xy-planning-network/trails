package auth

import "errors"

var (
	ErrNotValid   = errors.New("not valid")
	ErrUnexpected = errors.New("unexpected")
)
