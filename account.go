package trails

// An Account is a way many Users access a trails application
// and can be related to one another.
//
// An Account has many Users.
// An Account has one User designated as the owner of the Account.
type Account struct {
	Model
	AccessState    AccessState `json:"accessState"`
	AccountOwnerID int64       `json:"accountOwnerId"`

	// Associations
	AccountOwner *User  `json:"accountOwner,omitempty"`
	Users        []User `json:"users,omitempty"`
}
