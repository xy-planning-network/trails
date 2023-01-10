package trails

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xy-planning-network/trails/logger"
)

// An Environment is a different context in which a trails app operates.
type Environment string

const (
	Demo        Environment = "DEMO"
	Development Environment = "DEVELOPMENT"
	Production  Environment = "PRODUCTION"
	Review      Environment = "REVIEW"
	Staging     Environment = "STAGING"
	Testing     Environment = "TESTING"
)

func (e Environment) String() string { return string(e) }

func (e Environment) Valid() error {
	switch e {
	case Demo, Development, Production, Review, Staging, Testing:
		return nil
	default:
		return ErrNotValid
	}
}

func (e Environment) IsDevelopment() bool {
	return e == Development
}

func (e Environment) IsDemo() bool {
	return e == Demo
}

func (e Environment) IsProduction() bool {
	return e == Production
}

func (e Environment) IsReview() bool {
	return e == Review
}

func (e Environment) IsStaging() bool {
	return e == Staging
}

func (e Environment) IsTesting() bool {
	return e == Testing
}

// EnvVarOrBool gets the environment variable for the provided key and
// returns whether it matches "true" or "false" (after lower casing it)
// or the default value.
func EnvVarOrBool(key string, def bool) bool {
	val := os.Getenv(key)
	if strings.ToLower(val) == "true" {
		return true
	}

	if strings.ToLower(val) == "false" {
		return false
	}

	return def
}

// EnvVarOrDuration gets the environment variable for the provided key,
// parses it into a time.Duration, or, returns
// the default time.Duration.
func EnvVarOrDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}

// EnvVarOrEnv gets the environment variable for the provided key,
// casts it into an Environment,
// or returns the provided default Environment if key is not a valid Environment.
func EnvVarOrEnv(key string, def Environment) Environment {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	env := Environment(strings.ToUpper(val))
	if err := env.Valid(); err != nil {
		return def
	}

	return env
}

// EnvVarOrLogLevel gets the environment variable for the provided key,
// creates a logger.LogLevel from the retrieved value,
// or returns the provided default logger.LogLevel
// if the value is an unknown logger.LogLevel.
func EnvVarOrLogLevel(key string, def logger.LogLevel) logger.LogLevel {
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

// EnvVarOrString gets the environment variable for the provided key or the provided default string.
func EnvVarOrString(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	return val
}

// EnvVarOrURL gets the environment variable for the provided or the provided default *url.URL.
func EnvVarOrURL(key string, def *url.URL) *url.URL {
	if def.Path != "/" {
		def.Path = "/"
	}

	val := os.Getenv(key)
	if val == "" {
		return def
	}

	u, err := url.ParseRequestURI(val)
	if err != nil {
		return def
	}

	if u.Path != "/" {
		u.Path = "/"
	}

	return u
}
