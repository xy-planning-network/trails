package trails

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// A Model is the essential data points for primary ID-based models in a trails application,
// indicating when a record was created, last updated and soft deleted.
type Model struct {
	ID        uint        `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	DeletedAt DeletedTime `json:"deletedAt"`
}

// DeletedTime is a nullable timestamp marking a record as soft deleted.
type DeletedTime struct {
	sql.NullTime
}

// IsDeleted asserts whether the record is soft deleted.
func (dt DeletedTime) IsDeleted() bool { return !dt.Valid }

// Implements GORM-specific interfaces for modifying queries when DeletedTime is valid
// cf.:
// - https://github.com/go-gorm/gorm/blob/8dde09e0becd383bc24c7bd7d17e5600644667a8/soft_delete.go
func (DeletedTime) DeleteClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{gorm.SoftDeleteDeleteClause{Field: f}}
}
func (DeletedTime) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{gorm.SoftDeleteQueryClause{Field: f}}
}

func (DeletedTime) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{gorm.SoftDeleteUpdateClause{Field: f}}
}

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
