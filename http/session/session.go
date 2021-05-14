package session

import "net/http"

type Sessionable interface {
	Delete(w http.ResponseWriter, r *http.Request) error
	DeregisterUser(w http.ResponseWriter, r *http.Request) error
	// Flashes(w http.ResponseWriter, r *http.Request) []Flash // TODO(dlk): flesh out Flash first
	FetchFlashes(w http.ResponseWriter, r *http.Request) []interface{}
	Get(key string) interface{}
	RegisterUser(w http.ResponseWriter, r *http.Request, ID uint) error
	Save(w http.ResponseWriter, r *http.Request) error
	// SetFlash(w http.ResponseWriter, r *http.Request, flash Flash) // TODO(dlk): flesh out Flash first
	Set(w http.ResponseWriter, r *http.Request, key string, val interface{}) error
	SetFlash(w http.ResponseWriter, r *http.Request, class, message string)
	UserID() (uint, error)
}
