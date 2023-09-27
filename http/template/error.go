package template

import "errors"

var (
	ErrMatchedAssets = errors.New("multiple assets matched")
	ErrNoFiles       = errors.New("no files provided")
)
