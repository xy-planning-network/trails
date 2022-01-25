package domain

type Account struct {
	Model
	AccessState AccessState

	Users []User
}
