package postgres_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/postgres"
)

type Account struct {
	trails.Model
	Kind string
}

type User struct {
	trails.Model
	AccountID int64
	Email     string
	Name      string
	Role      string

	Account Account
}

func newUser(acctID int64) User {
	eid := uuid.New()
	parts := strings.Split(eid.String(), "-")
	return User{
		AccountID: acctID,
		Email:     parts[0] + "@example.com",
		Name:      parts[1] + " " + parts[2],
	}
}

type Login struct {
	trails.Model
	At     time.Time
	UserID int64

	User User
}

func connect(t *testing.T) *postgres.DB {
	t.Helper()

	cfg := postgres.Config{
		Env:      trails.Testing,
		IsTestDB: true,
		Host:     "localhost",
		Port:     "5432",
		Name:     "trails_test", // TODO
		User:     "davidketch",  // TODO
		Schema:   "public",      // TODO
	}
	db, err := postgres.Connect(cfg)
	require.Nil(t, err)

	b, err := os.ReadFile("testdata/schema.sql")
	require.Nil(t, err)

	err = db.Exec(string(b))
	require.Nil(t, err)

	return db
}

func insertAccounts(t *testing.T, db *postgres.DB) {
	t.Helper()
	accts := []Account{
		{Kind: "special"},
		{Kind: "exceptional"},
		{Kind: "exceptional"},
		{Kind: "default"},
		{Kind: "default"},
	}
	err := db.Create(&accts)
	require.Nil(t, err)
}

func insertUsers(t *testing.T, db *postgres.DB, accts []Account) {
	t.Helper()
	var users []User
	for _, acct := range accts {
		user := newUser(acct.ID)
		user.Role = "owner"
		users = append(users, user)
	}

	for _, acct := range accts {
		user := newUser(acct.ID)
		user.Role = "manager"
		users = append(users, user)
	}

	err := db.Create(&users)
	require.Nil(t, err)
}
