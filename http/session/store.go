package session

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/boj/redistore"
	gorilla "github.com/gorilla/sessions"
)

const defaultMaxAge = 86400 // 1 day

// The SessionStorer defines methods for interacting with a Sessionable for the given *http.Request.
type SessionStorer interface {
	GetSession(r *http.Request) (Sessionable, error) // TODO(dlk): Sessionable or TrailsSessionable?
}

// A Service wraps a gorilla.Store to manage constructing a new one
// and accessing the sessions contained in it.
//
// Service implements SessionStorer.
type Service struct {
	ak     []byte
	ek     []byte
	env    string
	maxAge int
	store  gorilla.Store
}

// NewStoreService initiates a data store for user web sessions
// with the provided hex-encoded authentication key and encryption keys.
// If no backing storage is provided through a functional option -
// like WithRedis - NewService stores sessions in cookies.
func NewStoreService(env, authKey, encryptKey string, opts ...ServiceOpt) (Service, error) {
	gob.Register(Flash{})
	var err error
	s := Service{env: env, maxAge: defaultMaxAge}

	s.ak, err = hex.DecodeString(authKey)
	if err != nil {
		return Service{}, err
	}
	s.ek, err = hex.DecodeString(encryptKey)
	if err != nil {
		return Service{}, err
	}

	for _, opt := range opts {
		if err := opt(&s); err != nil {
			return Service{}, fmt.Errorf("%w: %s", ErrFailedConfig, err)
		}
	}

	if s.store == nil {
		if err := WithCookie()(&s); err != nil {
			return Service{}, fmt.Errorf("%w: %s", ErrFailedConfig, err)
		}
	}

	return s, nil
}

// GetSession wraps gorilla.Get, creating a brand new Session or one from the session retrieved.
func (s Service) GetSession(r *http.Request) (Sessionable, error) {
	session, err := s.store.Get(r, sessionKey)
	return Session{session}, err
}

// A ServiceOpt configures the provided *Service,
// returning an error if unable to.
type ServiceOpt func(*Service) error

// WithCookie configures the Service to back session storage with cookies.
func WithCookie() ServiceOpt {
	var c *gorilla.CookieStore
	return func(s *Service) error {
		if !strings.EqualFold(s.env, "testing") {
			c = gorilla.NewCookieStore(s.ak, s.ek)
		} else {
			c = gorilla.NewCookieStore(s.ak)
		}
		c.Options.Secure = !strings.EqualFold(s.env, "development")
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
			return fmt.Errorf("%w: failed initializing Redis: %s", ErrFailedConfig, err)
		}
		r.Options.Secure = !strings.EqualFold(s.env, "development")
		r.Options.HttpOnly = true
		r.SetMaxAge(s.maxAge)
		s.store = r
		return nil
	}
}
