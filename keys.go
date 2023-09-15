package trails

type Key string

const (
	// AppPropsKey stashes additional props to be included in HTTP responses.
	AppPropsKey Key = "AppPropsKey"

	// CurrentUserKey stashes the currentUser for a session.
	CurrentUserKey Key = "CurrentUserKey"

	// IpAddrKey stashes the IP address of an HTTP request being handled by trails.
	IpAddrKey Key = "IpAddrKey"

	// RequestIDKey stashes a unique UUID for each HTTP request.
	RequestIDKey Key = "RequestIDKey"

	// SessionKey stashes the session associated with an HTTP request.
	SessionKey Key = "SessionKey"

	// SessionIDKey stashes a unique UUID for each session.
	SessionIDKey Key = "SessionIDKey"
)

// String formats the stringified key with additional contextual information
func (k Key) String() string {
	return "trails context key: " + string(k)
}
