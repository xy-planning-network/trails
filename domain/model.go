package domain

import (
	"database/sql"
	"fmt"
	"time"
)

type Model struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt DeletedTime
}

type DeletedTime struct {
	sql.NullTime
}

func (dt DeletedTime) IsDeleted() bool { return !dt.Valid }

type AccessState string

const (
	AccessGranted     AccessState = "granted"
	AccessInvited     AccessState = "invited"
	AccessRevoked     AccessState = "revoked"
	AccessVerifyEmail AccessState = "verify-email"
)

func (as AccessState) String() string { return string(as) }

func (as AccessState) Valid() error {
	switch as {
	case AccessGranted, AccessInvited, AccessRevoked, AccessVerifyEmail:
		return nil
	default:
		return fmt.Errorf("%w: AccessState %s", ErrNotValid, as.String())
	}
}
