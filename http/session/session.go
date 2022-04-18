package session

import (
	"net/http"

	gorilla "github.com/gorilla/sessions"
)

// keys used internal to specific implementations of different interfaces.
const (
	sessionKey     = "trails-session-gorilla" // used by Service
	userSessionKey = sessionKey + "-user"     // used by Session
)

// The Sessionable wraps methods for basic adding values to, deleting, and getting values from a session
// associated with an *http.Request and saving those to the session store.
type Sessionable interface {
	Delete(w http.ResponseWriter, r *http.Request) error
	Get(key string) any
	ResetExpiry(w http.ResponseWriter, r *http.Request) error
	Save(w http.ResponseWriter, r *http.Request) error
	Set(w http.ResponseWriter, r *http.Request, key string, val any) error
}

// The UserSessionable wraps methods for adding, removing, and retrieving
// user IDs from a session.
type UserSessionable interface {
	DeregisterUser(w http.ResponseWriter, r *http.Request) error
	RegisterUser(w http.ResponseWriter, r *http.Request, ID uint) error
	UserID() (uint, error)
}

// The TrailsSessionable composes session's major interfaces.
type TrailsSessionable interface {
	FlashSessionable
	Sessionable
	UserSessionable
}

// A Session provides all functionality for managing a fully featured session.
//
// Its functionality is implemented by lightly wrapping a gorilla.Session.
//
// TODO(dlk): embed *gorilla.Session anonymously? do not export?
type Session struct {
	s *gorilla.Session
}

// NewSession constructs a new Session as an implementation of TrailsSessionable
// from an interface value that is a *gorilla.Session.
// If it isn't, ErrNotValid returns.
//
// Typical usage is to pass in the value retrieved from a http.Request.Context.
// Given context keys are unexported, this package cannot perform that retrieval.
func NewSession(g *gorilla.Session) TrailsSessionable { return Session{s: g} }

func (s Session) ClearFlashes(w http.ResponseWriter, r *http.Request) {
	_ = s.Flashes(w, r)
	return
}

// Delete removes a session by making the MaxAge negative.
func (s Session) Delete(w http.ResponseWriter, r *http.Request) error {
	s.s.Options.MaxAge = -1
	return s.Save(w, r)
}

// DeregisterUser removes the User from the session.
func (s Session) DeregisterUser(w http.ResponseWriter, r *http.Request) error {
	delete(s.s.Values, userSessionKey)
	return s.Save(w, r)
}

// Flashes retrieves []Flash stored in the session.
func (s Session) Flashes(w http.ResponseWriter, r *http.Request) []Flash {
	raw := s.s.Flashes()
	fs := make([]Flash, 0)
	for _, r := range raw {
		f, ok := r.(Flash)
		if !ok {
			continue
		}

		fs = append(fs, f)
	}
	if len(fs) > 0 {
		// NOTE(dlk): Flashes are removed after they are accessed,
		// but the session needs to be saved for them to be finally removed
		if err := s.Save(w, r); err != nil {
			return nil
		}
	}

	return fs
}

// Get retrieves a value from the session according to the key passed in.
func (s Session) Get(key string) any {
	return s.s.Values[key]
}

// RegisterUserSession stores the user's ID in the session.
func (s Session) RegisterUser(w http.ResponseWriter, r *http.Request, ID uint) error {
	s.s.Values[userSessionKey] = ID
	return s.Save(w, r)
}

// ResetExpiry resets the expiration of the session by saving it.
func (s Session) ResetExpiry(w http.ResponseWriter, r *http.Request) error {
	return s.Save(w, r)
}

// Save wraps gorilla.Session.Save, saving the session in the request.
func (s Session) Save(w http.ResponseWriter, r *http.Request) error { return s.s.Save(r, w) }

// Set stores a value according to the key passed in on the session.
func (s Session) Set(w http.ResponseWriter, r *http.Request, key string, val any) error {
	s.s.Values[key] = val
	return s.Save(w, r)
}

// SetFlash stores the passed in Flash in the session.
func (s Session) SetFlash(w http.ResponseWriter, r *http.Request, flash Flash) error {
	s.s.AddFlash(flash)
	return s.Save(w, r)
}

// UserID gets the user ID out of the session.
// A user ID should be present in a session if the user is successfully authenticated.
// If no user ID can be found, this ErrNoUser is returned.
// This ought to only happen when a user is going through an authentication workflow or hitting unauthenticated pages.
//
// If the value returned from the session is not a uint, ErrNotValid is returned and represents a programming error.
func (s Session) UserID() (uint, error) {
	intfVal, ok := s.s.Values[userSessionKey]
	if !ok {
		return 0, ErrNoUser
	}

	val, ok := intfVal.(uint)
	if !ok {
		return 0, ErrNotValid
	}

	return val, nil
}

var _ TrailsSessionable = Stub{}

type Stub struct{}

func (s Stub) ClearFlashes(w http.ResponseWriter, r *http.Request)                {}
func (s Stub) Flashes(w http.ResponseWriter, r *http.Request) []Flash             { return nil }
func (s Stub) SetFlash(w http.ResponseWriter, r *http.Request, flash Flash) error { return nil }
func (s Stub) Delete(w http.ResponseWriter, r *http.Request) error                { return nil }
func (s Stub) Get(key string) any                                                 { return nil }
func (s Stub) ResetExpiry(w http.ResponseWriter, r *http.Request) error           { return nil }
func (s Stub) Save(w http.ResponseWriter, r *http.Request) error                  { return nil }
func (s Stub) Set(w http.ResponseWriter, r *http.Request, key string, val any) error {
	return nil
}
func (s Stub) DeregisterUser(w http.ResponseWriter, r *http.Request) error        { return nil }
func (s Stub) RegisterUser(w http.ResponseWriter, r *http.Request, ID uint) error { return nil }
func (s Stub) UserID() (uint, error)                                              { return 0, nil }
