package domain

import uuid "github.com/satori/go.uuid"

type User struct {
	Model
	AccessState AccessState
	AccountID   uint
	Email       string
	ExternalID  uuid.UUID

	Account *Account
}

func (u User) HasAccess() bool {
	if u.Account != nil {
		return u.Account.AccessState == AccessGranted && u.AccessState == AccessGranted
	}

	return u.AccessState == AccessGranted
}

func (u User) HomePath() string {
	if !u.HasAccess() {
		return "/login"
	}

	return "/"
}
