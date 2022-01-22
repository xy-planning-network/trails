package main

import (
	"errors"
	"net/http"

	"github.com/xy-planning-network/trails/http/keyring"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/http/session"
)

// registerUser mocks initializizing an authenticated user's session for example purposes
func registerUser(key keyring.Keyable, userstore *users) middleware.Adapter {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.URL.Query().Get("name")
			if name != "" {
				s, ok := r.Context().Value(key).(session.UserSessionable)
				if ok {
					u := append(*userstore, user{name})
					*userstore = u
					s.RegisterUser(w, r, uint(len(*userstore)-1))
				}
			}
			handler.ServeHTTP(w, r)
		})
	}
}

type users []middleware.User

func (u users) GetByID(id uint) (middleware.User, error) {
	if int(id) > len(u)-1 {
		return nil, errors.New("not found")
	}

	return u[id], nil
}

type user struct {
	Name string
}

func (u user) HomePath() string { return "/user" }
func (u user) HasAccess() bool  { return true }
