package postgres_test

import (
	"slices"
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

	Users []User
}

type User struct {
	trails.Model
	AccountID int64
	Email     string
	Name      string
	Role      string

	Account Account
	Logins  []Login
	Groups  []GroupUser
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

type Group struct {
	trails.Model
	Title string

	Users []GroupUser
}

type GroupUser struct {
	GroupID int64
	UserID  int64

	Group Group
	User  User
}

func insertAccounts(t *testing.T, db *postgres.DB) []Account {
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

	return accts
}

func insertUsers(t *testing.T, db *postgres.DB, accts []Account) []User {
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

	return users
}

func (suite *DBTestSuite) TestBegin() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestCount() {
	// Arrange + Act
	count, err := suite.db.Count()

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)
	suite.Require().Zero(count)

	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	// Act
	count, err = suite.db.Model(new(Account)).Count()

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(int64(5), count)
}

func (suite *DBTestSuite) TestCommit() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestCreate() {
	// Arrange
	notAPointer := Group{Title: "Test"}

	// Act
	err := suite.db.Create(notAPointer)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnaddressable)

	// Arrange + Act
	err = suite.db.Create(nil)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnaddressable)

	// Arrange
	acctFirst := Account{Kind: "test"}

	// Act
	err = suite.db.Create(&acctFirst)

	// Assert
	suite.Require().Nil(err)
	suite.Require().NotZero(acctFirst.ID)
	suite.Require().NotZero(acctFirst.CreatedAt)

	// Arrange
	var gu GroupUser

	// Act
	err = suite.db.Create(&gu)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	tx := suite.db.Begin()
	suite.Require().ErrorIs(
		tx.Exec("ALTER TABLE accounts ADD CONSTRAINT uniq_kind UNIQUE(kind)"),
		trails.ErrNotFound,
	)

	acctNotUniq := Account{Kind: "test"}

	// Act
	err = tx.Create(&acctNotUniq)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrExists)
	suite.Require().Nil(tx.Rollback())

	// Arrange
	noTable := new(struct{})

	// Act
	err = suite.db.Create(noTable)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)
}

func (suite *DBTestSuite) TestDelete() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestExec() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestExists() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestFind() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestFirst() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestGroup() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestJoins() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestLimit() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestModel() {
	// Arrange
	actual := insertAccounts(suite.T(), suite.db)

	var accts []struct {
		trails.Model
		Kind string
	}

	// Act
	err := suite.db.Model(new(Account)).Find(&accts)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(accts, 5)

	// Arrange
	_ = insertUsers(suite.T(), suite.db, actual)
	var users []struct {
		trails.Model
		AccountID int64
		Email     string
		Name      string
		Role      string
	}

	// Act
	err = suite.db.Model(new(Account)).Model(new(Group)).Model(new(User)).Find(&users)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(users, 10)
}

func (suite *DBTestSuite) TestOffset() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestOr() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestOrder() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestPaged() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestPreload() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	_ = insertUsers(suite.T(), suite.db, accts)

	actualU := new(User)

	// Act
	err := suite.db.Preload("Account").
		Where("account_id = ?", accts[0].ID).
		Where("role = ?", "owner").
		First(actualU)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[0].ID, actualU.AccountID)
	suite.Require().Equal(accts[0].ID, actualU.Account.ID)

	// Arrange
	actualA := new(Account)

	adminScope := func(dbx *postgres.DB) *postgres.DB { return dbx.Where("users.role = ?", "owner") }

	// Act
	err = suite.db.Preload("Users", adminScope).Where("id = ?", accts[0].ID).First(actualA)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[0].ID, actualA.ID)
	suite.Require().Len(actualA.Users, 1)
	suite.Require().Equal(actualU.ID, actualA.Users[0].ID)

	// Arrange
	actualU = new(User)
	suite.Require().Nil(suite.db.Where("account_id = ?", actualA.ID).Where("role = ?", "manager").First(actualU))

	actualA = new(Account)

	managerScope := func(dbx *postgres.DB) *postgres.DB { return dbx.Where("users.role = ?", "manager") }
	emailScope := func(dbx *postgres.DB) *postgres.DB { return dbx.Where("users.email ILIKE ?", "%example.com%") }

	// Act
	err = suite.db.Preload("Users", managerScope, emailScope).Where("id = ?", accts[0].ID).First(actualA)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[0].ID, actualA.ID)
	suite.Require().Len(actualA.Users, 1)
	suite.Require().Equal(actualU.ID, actualA.Users[0].ID)
}

func (suite *DBTestSuite) TestRaw() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestScope() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	var exceptional []Account
	for _, acct := range accts {
		if acct.Kind == "exceptional" {
			exceptional = append(exceptional, acct)
		}
	}

	exceptionalScope := func(dbx *postgres.DB) *postgres.DB {
		return dbx.Where("accounts.kind = ?", "exceptional")
	}

	var actualA []Account

	// Act
	err := suite.db.Scope(exceptionalScope).Find(&actualA)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(len(exceptional), len(actualA))

	// Arrange
	users := insertUsers(suite.T(), suite.db, accts)
	var exceptionalManagers []User
	for _, user := range users {
		isManager := user.Role == "manager"
		isExceptionalAcct := slices.ContainsFunc(exceptional, func(acct Account) bool {
			return acct.ID == user.AccountID
		})

		if isManager && isExceptionalAcct {
			exceptionalManagers = append(exceptionalManagers, user)
		}
	}

	managerScope := func(dbx *postgres.DB) *postgres.DB { return dbx.Where("users.role = ?", "manager") }

	var actualU []User

	// Act
	err = suite.db.Scope(exceptionalScope).Scope(managerScope).Find(&actualU)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(len(exceptionalManagers), len(actualU))
}

func (suite *DBTestSuite) TestSelect() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestTable() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestUnscoped() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestWhere() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestRollback() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestUpdate() {
	suite.T().FailNow()
}
