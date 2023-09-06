package trails

import (
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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

// CanUseServiceStub asserts whether the Environment allows for setting up with stubbed out services,
// for those services that support stubbing.
func (e Environment) CanUseServiceStub() bool {
	switch e {
	case Demo, Development, Testing:
		return true
	default:
		return false
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

// ToolboxEnabled asserts whether the Environment enables the client-side toolbox.
func (e Environment) ToolboxEnabled() bool {
	switch e {
	case Demo, Development, Staging, Testing:
		return true
	default:
		return false
	}
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
// parses it into a [time.Duration], or, returns
// the default [time.Duration].
func EnvVarOrDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}

// EnvVarOrEnv gets the environment variable for the provided key,
// casts it into an [Environment],
// or returns the provided default [Environment] if key is not a valid [Environment].
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

// EnvVarOrInt gets the environment variable for the provided key,
// creates an int from the retrieved value,
// or returns the provided default
// if the value is not a valid int.
func EnvVarOrInt(key string, def int) int {
	val, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}

	return val
}

// EnvVarOrLogLevel gets the environment variable for the provided key,
// creates a [log/slog.Level] from the retrieved value,
// or returns the provided default [log/slog.Level].
func EnvVarOrLogLevel(key string, def slog.Level) slog.Level {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	return NewLogLevel(val)
}

// EnvVarOrString gets the environment variable for the provided key or the provided default string.
func EnvVarOrString(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	return val
}

// EnvVarOrURL gets the environment variable for the provided or the provided default *url.URL.
func EnvVarOrURL(key, def string) *url.URL {
	defURL, err := url.ParseRequestURI(def)
	if err != nil {
		return nil
	}

	if defURL.Path != "/" {
		defURL.Path = "/"
	}

	val := os.Getenv(key)
	if val == "" {
		return defURL
	}

	u, err := url.ParseRequestURI(val)
	if err != nil {
		return defURL
	}

	return u
}
