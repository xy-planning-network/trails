package session

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/boj/redistore"
	gorilla "github.com/gorilla/sessions"
	"github.com/xy-planning-network/trails"
)

const defaultMaxAge = 86400 // 1 day

// The SessionStorer defines methods for interacting with a Sessionable for the given *http.Request.
type SessionStorer interface {
	GetSession(r *http.Request) (Session, error)
}

// A Service wraps a gorilla.Store to manage constructing a new one
// and accessing the sessions contained in it.
//
// Service implements SessionStorer.
type Service struct {
	// The authentication key.
	ak []byte

	// The encryption key.
	ek []byte

	// The name this Service's sessions are stored under.
	// Also used as the name of the cookie when WithCookie is used.
	sn string

	// The environment the Service is operating within.
	env trails.Environment

	// The number of seconds a session is valid.
	maxAge int

	// how the Service actually implements storing sessions.
	store gorilla.Store
}

// A Config provides the required values
type Config struct {
	Env trails.Environment

	// The name sessions are stored under.
	// Also used as the name of the cookie when WithCookie is used.
	SessionName string

	// Hex-encoded key
	AuthKey string

	// Hex-encoded key
	EncryptKey string
}

func validateConfig(c Config) error {
	err := c.Env.Valid()
	if err != nil {
		return err
	}

	if c.SessionName == "" {
		return fmt.Errorf("%w: SessionName cannot be %q", trails.ErrBadConfig, c.SessionName)
	}

	return nil
}

// NewStoreService initiates a data store for user web sessions
// with the provided config.
// If no backing storage is provided through a functional option -
// like WithRedis - NewService stores sessions in cookies.
func NewStoreService(cfg Config, opts ...ServiceOpt) (Service, error) {
	var err error
	gob.Register(Flash{})

	s := Service{
		env:    cfg.Env,
		maxAge: defaultMaxAge,
		sn:     cfg.SessionName,
	}

	s.ak, err = hex.DecodeString(cfg.AuthKey)
	if err != nil {
		return Service{}, fmt.Errorf("%w: authentication key is not valid: %s", trails.ErrBadConfig, err)
	}

	s.ek, err = hex.DecodeString(cfg.EncryptKey)
	if err != nil {
		return Service{}, fmt.Errorf("%w: encryption key is not valid: %s", trails.ErrBadConfig, err)
	}

	for _, opt := range opts {
		if err := opt(&s); err != nil {
			return Service{}, fmt.Errorf("%w: %s", trails.ErrBadConfig, err)
		}
	}

	if s.store == nil {
		if err := WithCookie()(&s); err != nil {
			return Service{}, fmt.Errorf("%w: %s", trails.ErrBadConfig, err)
		}
	}

	return s, nil
}

// GetSession retrieves the Session for the *http.Request,
// or creates a brand new one.
func (s Service) GetSession(r *http.Request) (Session, error) {
	session, err := s.store.Get(r, s.sn)
	return Session{s: session}, err
}

// A ServiceOpt configures the provided *Service,
// returning an error if unable to.
type ServiceOpt func(*Service) error

// WithCookie configures the Service to back session storage with cookies.
func WithCookie() ServiceOpt {
	var c *gorilla.CookieStore
	return func(s *Service) error {
		if !s.env.IsTesting() {
			c = gorilla.NewCookieStore(s.ak, s.ek)
		} else {
			c = gorilla.NewCookieStore(s.ak)
		}

		c.Options.Secure = !(s.env.IsDevelopment() || s.env.IsTesting())
		c.Options.HttpOnly = true
		c.MaxAge(s.maxAge)
		s.store = c
		return nil
	}
}

// WithMaxAge sets the time-to-live of a session.
//
// Call before other options so this value is available.
//
// Otherwise, the Service uses defaultMaxAge.
func WithMaxAge(secs int) ServiceOpt {
	return func(s *Service) error {
		s.maxAge = secs
		return nil
	}
}

// WithRedis configures the Service to back session storage with Redis.
//
// To authenticate to the Redis server, provide pass, otherwise its zero-value is acceptable.
func WithRedis(uri, pass string) ServiceOpt {
	var r *redistore.RediStore
	var err error
	return func(s *Service) error {
		if pass == "" {
			r, err = redistore.NewRediStore(10, "tcp", uri, "", s.ak, s.ek)
		} else {
			r, err = redistore.NewRediStore(10, "tcp", uri, pass, s.ak, s.ek)
		}
		if err != nil {
			return fmt.Errorf("%w: failed initializing Redis: %s", trails.ErrBadConfig, err)
		}
		r.Options.Secure = !(s.env.IsDevelopment() || s.env.IsTesting())
		r.Options.HttpOnly = true
		r.SetMaxAge(s.maxAge)
		s.store = r
		return nil
	}
}

type Stub struct {
	s *gorilla.Session
}

func NewStub(loggedIn bool) *Stub {
	s := new(Stub)
	s.s = gorilla.NewSession(s, "stub")
	if loggedIn {
		s.s.Values[trails.CurrentUserKey] = uint(1)
	}

	return s
}

func (s *Stub) GetSession(r *http.Request) (Session, error) {
	return Session{s.s}, nil

}

func (s Stub) Get(r *http.Request, name string) (*gorilla.Session, error)               { return s.s, nil }
func (s Stub) New(r *http.Request, name string) (*gorilla.Session, error)               { return s.s, nil }
func (s Stub) Save(r *http.Request, w http.ResponseWriter, sess *gorilla.Session) error { return nil }
