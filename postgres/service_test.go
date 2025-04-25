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

func connect(t *testing.T) {
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
	err := postgres.Connect(cfg)
	require.Nil(t, err)

	b, err := os.ReadFile("testdata/schema.sql")
	require.Nil(t, err)

	_, err = postgres.Query[struct{}]().Exec(string(b))
	require.Nil(t, err)

	return
}

func insertAccounts(t *testing.T) {
	t.Helper()
	accts := []Account{
		{Kind: "special"},
		{Kind: "exceptional"},
		{Kind: "exceptional"},
		{Kind: "default"},
		{Kind: "default"},
	}
	err := postgres.Query[[]Account]().Create(accts)
	require.Nil(t, err)
}

func insertUsers(t *testing.T, accts []Account) {
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

	err := postgres.Query[[]User]().Create(users)
	require.Nil(t, err)
}

func TestQueryDistinct(t *testing.T) {
	// Arrange
	connect(t)
	insertAccounts(t)

	// Act
	accounts, err := postgres.Query[Account]().Distinct("kind").Find()

	// Assert
	require.Nil(t, err)
	require.Len(t, accounts, 3)
}

func TestQueryFind(t *testing.T) {
	// Arrange
	connect(t)

	// Act
	none, err := postgres.Query[postgres.None]().Find()

	// Assert
	require.ErrorIs(t, err, trails.ErrUnexpected)
	require.Zero(t, none)

	// Arrange + Act
	accounts, err := postgres.Query[Account]().Find()

	// Arrange
	insertAccounts(t)

	// Act
	accounts, err = postgres.Query[Account]().Find()

	// Assert
	require.Nil(t, err)
	require.Len(t, accounts, 5)

	// Arrange
	insertUsers(t, accounts)

	// Act
	users, err := postgres.Query[User]().Find()

	// Assert
	require.Nil(t, err)
	require.Len(t, users, 10)
}

func TestQueryWhere(t *testing.T) {
	// Arrange
	connect(t)
	insertAccounts(t)

	// Act
	accounts, err := postgres.Query[Account]().Where("kind = ?", "does not exist").Find()

	// Assert
	require.ErrorIs(t, err, trails.ErrNotFound)
	require.Zero(t, accounts)

	// Act
	accounts, err = postgres.Query[Account]().Where("kind = ?", "default").Find()

	// Assert
	require.Nil(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, "default", accounts[0].Kind)
	require.Equal(t, "default", accounts[1].Kind)

	// Arrange
	allAccts, err := postgres.Query[Account]().Find()
	require.Nil(t, err)
	insertUsers(t, allAccts)

	// Act
	users, err := postgres.Query[User]().Where("role = ?", "owner").Find()

	// Assert
	require.Nil(t, err)
	require.Len(t, users, 5)
}

func TestQueryFirst(t *testing.T) {
	// Arrange
	connect(t)

	// Act
	account, err := postgres.Query[Account]().First()

	// Assert
	require.ErrorIs(t, err, trails.ErrNotFound)
	require.Zero(t, account)

	// Arrange
	insertAccounts(t)
	all, err := postgres.Query[Account]().Find()
	require.Nil(t, err)

	// Act
	account, err = postgres.Query[Account]().Order("id").First()

	// Assert
	require.Nil(t, err)
	require.Equal(t, all[0].ID, account.ID)
}
