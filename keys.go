package trails

import "context"

type Key string

const (
	// appPropsKey stashes additional props to be included in HTTP responses.
	appPropsKey Key = "AppPropsKey"

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

// An AppProps passes data from the server to the client as a set of props needed for general application state.
// The data is passed around in a context.Context and rendered as JSON.
// The data is expected to be marshaled into Vue/JS props.
//
// NB: Data not representable by JSON will create errors; review [encoding/json.Marshaler].
type AppProps map[string]any

// NewAppPropsContext adds props to ctx, returning the resulting context.
// If props have already been added to ctx, it's key-value pairs are added to existing ones.
// If any keys collide, those in props overwrite previous values.
func NewAppPropsContext(ctx context.Context, props AppProps) context.Context {
	existing := AppPropsFromContext(ctx)
	for k, v := range props {
		existing[k] = v
	}

	return context.WithValue(ctx, appPropsKey, existing)
}

// AppPropsFromContext retrieves an AppProps in ctx.
// If not already set, it initializes a new AppProps.
func AppPropsFromContext(ctx context.Context) AppProps {
	props, ok := ctx.Value(appPropsKey).(AppProps)
	if !ok {
		props = make(AppProps)
	}

	return props
}
