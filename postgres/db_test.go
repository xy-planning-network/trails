package postgres_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/postgres"
	"gorm.io/datatypes"
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
	ID     int64
	At     time.Time
	UserID int64

	User User
}

type Group struct {
	trails.Model
	Title  string
	Starts time.Time

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

func insertGroups(t *testing.T, db *postgres.DB) []Group {
	t.Helper()
	groups := []Group{
		{Title: "First", Starts: time.Now().AddDate(0, 0, -2)},
		{Title: "Second", Starts: time.Now().AddDate(0, 0, -1)},
		{Title: "Third", Starts: time.Now()},
	}
	err := db.Create(&groups)
	require.Nil(t, err)

	return groups
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
	// Arrange
	tx := suite.db.Begin()
	acct := Account{Kind: "commit-test"}
	suite.Require().Nil(tx.Create(&acct))
	suite.Require().NotZero(acct.ID)

	var actual Account

	// Act
	err := tx.Commit()

	// Assert
	suite.Require().Nil(err)
	suite.Require().Nil(suite.db.Where("id = ?", acct.ID).First(&actual))
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
	// Arrange
	var acct Account

	// Act
	err := suite.db.Where("id = ?", acct.ID).Delete(&acct)

	// Assert
	suite.Require().Nil(err)

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	var actual Account

	// Act
	err = suite.db.Delete(&accts[0])

	// Assert
	suite.Require().Nil(err)
	suite.Require().ErrorIs(
		suite.db.Where("id = ?", accts[0].ID).First(&actual),
		trails.ErrNotFound,
	)
}

func (suite *DBTestSuite) TestDistinct() {
	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	var actual []string

	// Act
	err := suite.db.Model(new(Account)).Distinct("kind").Select("kind").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().ElementsMatch([]string{"special", "exceptional", "default"}, actual)
}

func (suite *DBTestSuite) TestExec() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	q := "UPDATE accounts SET kind = 'exec-test' WHERE id = ?"

	// Act
	err := suite.db.Exec(q, accts[0].ID)

	// Assert
	suite.Require().Nil(err)

	var actual Account
	err = suite.db.Where("id = ?", accts[0].ID).First(&actual)
	suite.Require().Nil(err)
	suite.Require().Equal("exec-test", actual.Kind)
}

func (suite *DBTestSuite) TestExists() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	// Act
	actual, err := suite.db.Model(new(Account)).Where("id = ?", accts[0].ID).Exists()

	// Assert
	suite.Require().Nil(err)
	suite.Require().True(actual)

	// Arrange
	suite.Require().Nil(suite.db.Delete(&accts[0]))

	// Act
	actual, err = suite.db.Model(new(Account)).Where("id = ?", accts[0].ID).Exists()

	// Assert
	suite.Require().Nil(err)
	suite.Require().False(actual)

	// Arrange
	users := insertUsers(suite.T(), suite.db, accts)
	groups := insertGroups(suite.T(), suite.db)

	var ugs []GroupUser
	for _, user := range users {
		ugs = append(ugs, GroupUser{GroupID: groups[0].ID, UserID: user.ID})
	}

	for i := 0; i < 3; i++ {
		ugs = append(ugs, GroupUser{GroupID: groups[1].ID, UserID: users[i].ID})
	}

	suite.Require().Nil(suite.db.Create(&ugs))

	// Act
	actual, err = suite.db.Model(new(User)).
		Joins("JOIN group_users ON users.id = group_users.user_id").
		Where("group_id = ?", groups[0].ID).
		Where("user_id = ?", users[0].ID).
		Exists()

	// Assert
	suite.Require().Nil(err)
	suite.Require().True(actual)
}

func (suite *DBTestSuite) TestFind() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	var actual []Account

	// Act
	err := suite.db.Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, len(accts))

	// Arrange
	var kinds []string

	// Act
	err = suite.db.Model(new(Account)).Select("kind").Find(&kinds)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(kinds, len(accts))
}

func (suite *DBTestSuite) TestFirst() {
	// Arrange
	groups := insertGroups(suite.T(), suite.db)

	var actual Group

	// Act
	err := suite.db.Order("starts").First(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(groups[0].ID, actual.ID)

	// Arrange
	var actualGroups []Group

	// Act
	err = suite.db.Order("starts").First(&actualGroups)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualGroups, 1)
}

func (suite *DBTestSuite) TestGroup() {
	// Arrange

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
	// Arrange
}

func (suite *DBTestSuite) TestOr() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestOrder() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestPaged() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	users := insertUsers(suite.T(), suite.db, accts)

	// Act
	actual, err := suite.db.Model(new(User)).Paged(1, 2)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(int64(1), actual.Page)
	suite.Require().Equal(int64(2), actual.PerPage)
	suite.Require().Equal(int64(len(users)), actual.TotalItems)
	suite.Require().Equal(int64(5), actual.TotalPages)

	v, ok := actual.Items.(*[]User)
	suite.Require().True(ok)

	vv := *v
	suite.Require().Len(vv, 2)
	suite.Require().Equal(users[0].ID, vv[0].ID)
	suite.Require().Equal(users[1].ID, vv[1].ID)

	// Arrange + Act
	actual, err = suite.db.Model(new([]User)).Paged(2, 2)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(int64(2), actual.Page)
	suite.Require().Equal(int64(2), actual.PerPage)
	suite.Require().Equal(int64(len(users)), actual.TotalItems)
	suite.Require().Equal(int64(5), actual.TotalPages)

	v, ok = actual.Items.(*[]User)
	suite.Require().True(ok)

	vv = *v
	suite.Require().Len(vv, 2)
	suite.Require().Equal(users[2].ID, vv[0].ID)
	suite.Require().Equal(users[3].ID, vv[1].ID)

}

func (suite *DBTestSuite) TestPreload() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	users := insertUsers(suite.T(), suite.db, accts)
	expectedA := accts[0]
	expectedU := users[0]

	logins := []*Login{
		&Login{At: time.Now(), UserID: expectedU.ID},
		&Login{At: time.Now(), UserID: expectedU.ID},
		&Login{At: time.Now(), UserID: expectedU.ID},
		&Login{At: time.Now(), UserID: expectedU.ID},
		&Login{At: time.Now(), UserID: expectedU.ID},
	}
	suite.Require().Nil(suite.db.Create(&logins))

	actualU := new(User)

	// Act
	err := suite.db.
		Preload("Account").
		Preload("Logins").
		Where("account_id = ?", expectedA.ID).
		Where("role = ?", "owner").
		First(actualU)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(expectedA.ID, actualU.AccountID)
	suite.Require().Equal(expectedA.ID, actualU.Account.ID)
	suite.Require().Equal(len(logins), len(actualU.Logins))

	// Arrange
	actualA := new(Account)

	ownerScope := func(dbx *postgres.DB) *postgres.DB { return dbx.Where("users.role = ?", "owner") }

	// Act
	err = suite.db.Preload("Users", ownerScope).Where("id = ?", accts[0].ID).First(actualA)

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
	err = suite.db.Scope(exceptionalScope).
		Scope(managerScope).
		Joins("JOIN accounts ON accounts.id = users.account_id").
		Find(&actualU)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(len(exceptionalManagers), len(actualU))
}

func (suite *DBTestSuite) TestSelect() {
	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	var actual []string

	// Act
	err := suite.db.Model(new(Account)).Select("kind").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 5)
	suite.Require().Subset(actual, []string{"special", "exceptional", "default"})
}

func (suite *DBTestSuite) TestTable() {
	// Arrange
	tx := suite.db.Begin()
	tx.Exec("CREATE TABLE temp (col text)")
	tx.Exec("INSERT INTO temp (col) VALUES ('foo'), ('bar'), ('baz')")

	var actual []string

	// Act
	err := tx.Table("temp").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 3)
	suite.Require().Nil(tx.Rollback())
}

func (suite *DBTestSuite) TestUnscoped() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	suite.Require().Nil(suite.db.Delete(&accts[0]))

	actual := new(Account)

	// Act
	err := suite.db.Unscoped().Where("id = ?", accts[0].ID).First(actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[0].ID, actual.ID)
	suite.Require().True(actual.IsDeleted())
}

func (suite *DBTestSuite) TestWhere() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestRollback() {
	suite.T().FailNow()
}

func (suite *DBTestSuite) TestUpdate() {
	// Arrange
	acct := new(Account)
	suite.db.Create(acct)

	user := &User{AccountID: acct.ID, Name: "Jimothy Bobbitz"}
	suite.db.Create(user)

	updates := make(postgres.Updates)

	// Act
	err := suite.db.Model(new(User)).Where("id = ?", user.ID).Update(updates)

	// Assert
	suite.Require().Nil(err)

	// Arrange
	updates["name"] = "Bobbitz Jimothy"

	var actual User

	// Act
	err = suite.db.Model(new(User)).Where("id = ?", user.ID).Update(updates)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Nil(suite.db.Where("id = ?", user.ID).First(&actual))
	suite.Require().Equal(updates["name"], actual.Name)
}

type enum string

func (e enum) String() string { return string(e) }
func (e enum) Valid() error {
	switch e {
	case enumOne:
		return nil
	default:
		return fmt.Errorf("%w: %q", trails.ErrNotValid, e)
	}
}

const enumOne enum = "enum"

func (suite *DBTestSuite) TestStripNils() {
	var nilpqStringArray pq.StringArray
	nilDatatypesJSON := datatypes.JSON(json.RawMessage(`null`))

	// Arrange
	for _, tc := range []struct {
		input    postgres.Updates
		expected postgres.Updates
	}{
		{nil, nil},

		{
			postgres.Updates{
				"string": sql.NullString{},
				"number": sql.NullInt64{},
				"float":  sql.NullFloat64{},
				"bool":   sql.NullBool{},
				"byte":   sql.NullByte{},
			},
			make(postgres.Updates),
		},

		{
			postgres.Updates{
				"string": nil,
				"number": nil,
				"float":  nil,
				"bool":   nil,
				"byte":   nil,
			},
			make(postgres.Updates),
		},

		{
			postgres.Updates{
				"string": "just a string",
				"number": 12345,
				"float":  1.2345,
				"bool":   true,
				"byte":   []byte("\x00"),
			},
			postgres.Updates{
				"string": "just a string",
				"number": 12345,
				"float":  1.2345,
				"bool":   true,
				"byte":   []byte("\x00"),
			},
		},

		{
			postgres.Updates{
				"sql.NullString":     sql.NullString{Valid: true, String: "just a string"},
				"nil enum":           enum(""),
				"enumOne":            enumOne,
				"nil pq.StringArray": nilpqStringArray,
				"pq.StringArray":     pq.StringArray{"some", "strings"},
				"nil datatypes.JSON": nilDatatypesJSON,
				"datatypes.JSON":     datatypes.JSON(json.RawMessage(`"string"`)),
			},
			postgres.Updates{
				"sql.NullString": sql.NullString{Valid: true, String: "just a string"},
				"enumOne":        enumOne,
				"pq.StringArray": pq.StringArray{"some", "strings"},
				"datatypes.JSON": datatypes.JSON(json.RawMessage(`"string"`)),
			},
		},
	} {
		// Act
		tc.input.StripNils()

		// Assert
		suite.Require().Equal(tc.expected, tc.input)
	}
}
