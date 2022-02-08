package domain

import (
	"database/sql"
	"time"
)

// A Model is the essential data points for primary ID-based models in a trails application,
// indicating when a record was created, last updated and soft deleted.
type Model struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt DeletedTime
}

// DeletedTime is a nullable timestamp marking a record as soft deleted.
type DeletedTime struct {
	sql.NullTime
}

// IsDeleted asserts whether the record is soft deleted.
func (dt DeletedTime) IsDeleted() bool { return !dt.Valid }

// AccessState is a string representation of the broadest, general access
// an entity such as an Account or a User has to a trails application.
type AccessState string

const (
	AccessGranted     AccessState = "granted"
	AccessInvited     AccessState = "invited"
	AccessRevoked     AccessState = "revoked"
	AccessVerifyEmail AccessState = "verify-email"
)

// String stringifies the AccessState.
//
// String implements fmt.Stringer.
func (as AccessState) String() string { return string(as) }
