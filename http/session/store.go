package session

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	gorilla "github.com/gorilla/sessions"
	"github.com/xy-planning-network/trails"
)

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

	// how the Service actually implements storing sessions.
	store gorilla.Store
}

// A Config provides values required for constructing a session.Service.
type Config struct {
	// The domain to assign cookies to.
	Domain string

	Env trails.Environment

	// The number of seconds a session is valid.
	MaxAge int

	// The name sessions are stored under.
	// Also used as the name of the cookie when WithCookie is used.
	SessionName string

	// The SameSite mode for the session cookie.
	SameSiteMode http.SameSite

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

// NewStoreService initiates a data store for user web sessions with the provided config.
func NewStoreService(cfg Config) (Service, error) {
	var err error
	gob.Register(Flash{})
	gob.Register(trails.Key(""))

	s := Service{
		env: cfg.Env,
		sn:  cfg.SessionName,
	}

	s.ak, err = hex.DecodeString(cfg.AuthKey)
	if err != nil {
		return Service{}, fmt.Errorf("%w: authentication key is not valid: %s", trails.ErrBadConfig, err)
	}

	s.ek, err = hex.DecodeString(cfg.EncryptKey)
	if err != nil {
		return Service{}, fmt.Errorf("%w: encryption key is not valid: %s", trails.ErrBadConfig, err)
	}

	var c *gorilla.CookieStore
	if !s.env.IsTesting() {
		c = gorilla.NewCookieStore(s.ak, s.ek)
	} else {
		c = gorilla.NewCookieStore(s.ak)
	}

	c.Options.Domain = cfg.Domain
	c.Options.HttpOnly = true
	c.Options.SameSite = cfg.SameSiteMode
	c.Options.Secure = !(s.env.IsDevelopment() || s.env.IsTesting())
	c.MaxAge(cfg.MaxAge)

	s.store = c

	return s, nil
}

// GetSession retrieves the Session for the *http.Request,
// or creates a brand new one.
func (s Service) GetSession(r *http.Request) (Session, error) {
	session, err := s.store.Get(r, s.sn)
	if _, ok := session.Values[trails.SessionIDKey]; !ok {
		session.Values[trails.SessionIDKey] = uuid.NewString()
	}

	return Session{s: session}, err
}

// A ServiceOpt configures the provided *Service,
// returning an error if unable to.
type ServiceOpt func(*Service) error

type Stub struct {
	s *gorilla.Session
}

func NewStub(loggedIn bool) *Stub {
	s := new(Stub)
	s.s = gorilla.NewSession(s, "stub")
	if loggedIn {
		s.s.Values[trails.CurrentUserKey] = int64(1)
	}

	return s
}

func (s *Stub) GetSession(r *http.Request) (Session, error) {
	return Session{s.s}, nil

}

func (s Stub) Get(r *http.Request, name string) (*gorilla.Session, error)               { return s.s, nil }
func (s Stub) New(r *http.Request, name string) (*gorilla.Session, error)               { return s.s, nil }
func (s Stub) Save(r *http.Request, w http.ResponseWriter, sess *gorilla.Session) error { return nil }
