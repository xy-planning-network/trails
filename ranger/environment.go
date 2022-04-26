package ranger

import (
	"net/url"
	"os"
	"time"

	"github.com/xy-planning-network/trails/logger"
)

// An Environment is a different context in which a trails app operates.
type Environment string

const (
	Development Environment = "DEVELOPMENT"
	Production  Environment = "PRODUCTION"
	Review      Environment = "REVIEW"
	Staging     Environment = "STAGING"
	Testing     Environment = "TESTING"
)

func (e Environment) String() string { return string(e) }

func (e Environment) Valid() error {
	switch e {
	case Development, Production, Review, Staging, Testing:
		return nil
	default:
		return ErrNotValid
	}
}

// envVarOrDuration gets the environment variable from the provided key,
// parses it into a time.Duration, or, returns
// the default time.Duration.
func envVarOrDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}

// envVarOrEnv gets the environment variable from the provided key,
// casts it into an Environment,
// or returns the provided default Environment if key is not a valid Environment.
func envVarOrEnv(key string, def Environment) Environment {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	env := Environment(val)
	if err := env.Valid(); err != nil {
		return def
	}

	return env
}

// envVarOrLogLevel gets the environment variable from the provided key,
// creates a logger.LogLevel from the retrieved value,
// or returns the provided default logger.LogLevel
// if the value is an unknown logger.LogLevel.
func envVarOrLogLevel(key string, def logger.LogLevel) logger.LogLevel {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	ll := logger.NewLogLevel(val)
	if ll == logger.LogLevelUnk {
		return def
	}

	return ll
}

// envVarOrString gets the environment variable from the provided key or the provided default string.
func envVarOrString(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	return val
}

func envVarOrURL(key string, def *url.URL) *url.URL {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	u, err := url.ParseRequestURI(val)
	if err != nil {
		return def
	}

	return u
}
