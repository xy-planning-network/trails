package trails

import "github.com/google/uuid"

// A User is the core entity that interacts with a trails application.
//
// An agent's HTTP requests are authenticated first by a specific request
// with email & password data matching credentials stored on a DB record for a User.
// Upon a match, a session is created and stored.
// Further requests are authenticated by referencing that session.
//
// A User has one Account.
type User struct {
	Model
	AccessState AccessState `json:"accessState"`
	AccountID   uint        `json:"accountId"`
	Email       string      `json:"email"`
	ExternalID  uuid.UUID   `json:"externalId"`
	Password    []byte      `json:"-"`

	// Associations
	Account *Account `json:"account,omitempty"`
}

// HasAccess asserts whether the User's properties give it general
// access to the trails application.
func (u User) HasAccess() bool {
	if u.Account != nil {
		return u.Account.AccessState == AccessGranted && u.AccessState == AccessGranted
	}

	return u.AccessState == AccessGranted
}

// HomePath returns the relative URL path designated
// as the default resource in the trails applicaiton
// they can access.
func (u User) HomePath() string {
	if !u.HasAccess() {
		return "/login"
	}

	return "/"
}

func (u User) GetID() uint { return u.ID }
