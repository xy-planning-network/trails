package ranger

import "errors"

var (
	ErrBadConfig  = errors.New("bad config") // Configuration is invalid
	ErrNotExist   = errors.New("not exist")  // The entity requested does not exist
	ErrNotValid   = errors.New("invalid")    // The concrete value is not valid for its type
	ErrUnexpected = errors.New("unexpected") // The app encountered an unhandled error
)
