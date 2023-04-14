package session

import (
	"net/http"

	gorilla "github.com/gorilla/sessions"
	"github.com/xy-planning-network/trails"
)

// A Session provides all functionality for managing a fully featured session.
//
// Its functionality is implemented by lightly wrapping a gorilla.Session.
type Session struct {
	s *gorilla.Session
}

// ClearFlashes removes all Flashes from the Session.
func (s Session) ClearFlashes(w http.ResponseWriter, r *http.Request) {
	_ = s.Flashes(w, r)
}

// Delete removes a session by making the MaxAge negative.
func (s Session) Delete(w http.ResponseWriter, r *http.Request) error {
	s.s.Options.MaxAge = -1
	return s.Save(w, r)
}

// DeregisterUser removes the User from the session.
func (s Session) DeregisterUser(w http.ResponseWriter, r *http.Request) error {
	delete(s.s.Values, trails.CurrentUserKey)
	return s.Save(w, r)
}

// Flashes retrieves []Flash stored in the session.
func (s Session) Flashes(w http.ResponseWriter, r *http.Request) []Flash {
	raw := s.s.Flashes()
	if len(raw) == 0 {
		return nil
	}

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
	s.s.Values[trails.CurrentUserKey] = ID
	return s.Save(w, r)
}

// ResetExpiry resets the expiration of the session by saving it.
func (s Session) ResetExpiry(w http.ResponseWriter, r *http.Request) error {
	return s.Save(w, r)
}

// Save wraps gorilla.Session.Save, saving the session in the request.
func (s Session) Save(w http.ResponseWriter, r *http.Request) error { return s.s.Save(r, w) }

// Set stores a value according to the key passed in on the session.
func (s Session) Set(w http.ResponseWriter, r *http.Request, key trails.Key, val any) error {
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
	intfVal, ok := s.s.Values[trails.CurrentUserKey]
	if !ok {
		return 0, ErrNoUser
	}

	val, ok := intfVal.(uint)
	if !ok {
		return 0, ErrNotValid
	}

	return val, nil
}
