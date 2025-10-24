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
	"gorm.io/gorm"
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
	Status    string

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
	Title      string
	Starts     time.Time
	NullTime   sql.NullTime
	NullString sql.NullString

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
	last := len(accts) - 1
	for i, acct := range accts {
		user := newUser(acct.ID)
		user.Role, user.Status = "owner", "active"
		if i == last {
			user.Status = "inactive"
		}
		users = append(users, user)
	}

	for i, acct := range accts {
		user := newUser(acct.ID)
		user.Role, user.Status = "manager", "active"
		if i == last {
			user.Status = "inactive"
		}
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

	// Arrange
	count, err = suite.db.
		Model(new(Account)).
		Where("id = ?", 1, 2).
		Count()

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)
	suite.Require().Zero(count)
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

	// Arrange
	tx = suite.db.Begin()
	suite.Require().Nil(tx.Rollback())

	// Act
	err = tx.Commit()

	// Assert
	suite.Require().Error(err)
}

func (suite *DBTestSuite) TestCreate() {
	// Arrange
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	err := db.Create(nil)

	// Assert
	suite.Require().ErrorIs(err, testErr)

	// Arrange
	notAPointer := Group{Title: "Test"}

	// Act
	err = suite.db.Create(notAPointer)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnaddressable)

	// Arrange
	s := "just a string"

	// Act
	err = suite.db.Create(&s)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrMissingData)

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
	updates := postgres.Updates{"kind": "a-map"}

	// Act
	err = suite.db.Model(new(Account)).Create(updates)

	// Assert
	suite.Require().Nil(err)

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
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	err := db.Delete(nil)

	// Assert
	suite.Require().ErrorIs(err, testErr)

	// Arrange
	var notTable string

	// Act
	err = suite.db.Delete(notTable)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrMissingData)

	// Arrange
	var acct Account

	// Act
	err = suite.db.Where("id = ?", acct.ID).Delete(&acct)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotFound)

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

	// Arrange
	nonexistent := struct{ Name string }{Name: "test"}

	// Act
	err = suite.db.Delete(nonexistent)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)
}

func (suite *DBTestSuite) TestDistinct() {
	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	var actual []string

	// Act
	err := suite.db.Model(new(Account)).Distinct("").Select("kind").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().ElementsMatch([]string{"special", "exceptional", "default"}, actual)

	// Arrange
	actual = []string{}

	// Act
	err = suite.db.Model(new(Account)).Distinct("kind").Select("kind").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().ElementsMatch([]string{"special", "exceptional", "default"}, actual)
}

func (suite *DBTestSuite) TestExec() {
	// Arrange
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	err := db.Exec("")

	// Assert
	suite.Require().ErrorIs(err, testErr)

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	q := "UPDATE accounts SET kind = 'exec-test' WHERE id = ?"

	// Act
	err = suite.db.Exec(q, accts[0].ID)

	// Assert
	suite.Require().Nil(err)

	var actual Account
	err = suite.db.Where("id = ?", accts[0].ID).First(&actual)
	suite.Require().Nil(err)
	suite.Require().Equal("exec-test", actual.Kind)

	// Arrange
	q = "UPDATE accounts SET fake_column = 'exec-test' WHERE id = ?"

	// Act
	err = suite.db.Exec(q, accts[0].ID)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)
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

	// Arrange
	//var notAStruct any
	notAStruct := "just a string"

	// Act
	err = suite.db.Model(new(Account)).Find(&notAStruct)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	notAStruct2 := []int{}

	// Act
	err = suite.db.Model(new(Account)).Find(&notAStruct2)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)
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

// FIXME(dlk): The below captures unexpected behavior
// that I can't pin down to GORM or postgres.DB.
// It is not the desired behavior.
// I've looked at previous versions of GORM to see if there's a regression.
// Both packages show the same behavior.
//
// As seen in the validate identifier,
// First loads the expected values for sql.NullTime & sql.NullString
// and does not when re-using the group identifier that already has a valid value set.
func (suite *DBTestSuite) TestFirst_SQLNullBug() {
	// Arrange
	group := Group{
		Title:      "Broken sql.Null* postgres.DB",
		Starts:     time.Now(),
		NullTime:   sql.NullTime{Time: time.Now(), Valid: true},
		NullString: sql.NullString{String: "Broken", Valid: true},
	}
	suite.Require().Nil(suite.db.Create(&group))

	suite.Require().Nil(suite.db.Model(new(Group)).
		Where("id = ?", group.ID).
		Update(postgres.Updates{"null_time": nil, "null_string": nil}),
	)

	var validate Group
	suite.Require().Nil(suite.db.Where("id = ?", group.ID).First(&validate))
	suite.Require().False(validate.NullTime.Valid)
	suite.Require().False(validate.NullString.Valid)

	// Act
	err := suite.db.Where("id = ?", group.ID).First(&group)

	// Assert
	suite.Require().Nil(err)
	suite.Require().True(group.NullTime.Valid)
	suite.Require().True(group.NullString.Valid)

	// Arrange
	gdb := suite.db.DB()
	group = Group{
		Title:      "Broken sql.Null* GORM",
		Starts:     time.Now(),
		NullTime:   sql.NullTime{Time: time.Now(), Valid: true},
		NullString: sql.NullString{String: "Broken", Valid: true},
	}
	suite.Require().Nil(gdb.Create(&group).Error)

	suite.Require().Nil(gdb.Model(new(Group)).
		Where("id = ?", group.ID).
		Updates(map[string]any{"null_time": nil, "null_string": nil}).
		Error,
	)

	validate = Group{}
	suite.Require().Nil(gdb.Where("id = ?", group.ID).First(&validate).Error)
	suite.Require().False(validate.NullTime.Valid)
	suite.Require().False(validate.NullString.Valid)

	// Act
	err = gdb.Where("id = ?", group.ID).First(&group).Error

	// Assert
	suite.Require().Nil(err)
	suite.Require().True(group.NullTime.Valid)
	suite.Require().True(group.NullString.Valid)
}

func (suite *DBTestSuite) TestGroup() {
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	_ = insertUsers(suite.T(), suite.db, accts)

	var actual []struct {
		ID    int64 `gorm:"column:account_id"`
		Count int64 `gorm:"column:count"`
	}

	// Act
	err := suite.db.Group("account_id").
		Select("account_id AS account_id", "count(*) AS count").
		Order("account_id").
		Find(new([]User))

	// Assert
	suite.Require().Nil(err)
	for _, row := range actual {
		suite.Require().Equal(int64(2), row.Count)
	}
}

func (suite *DBTestSuite) TestJoins() {
	// Arrange + Act
	err := suite.db.Joins("").First(new([]Account))

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotFound)

	// Arrange + Act
	err = suite.db.Joins("not a real statement").First(new([]Account))

	// Asert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	users := insertUsers(suite.T(), suite.db, accts)
	login := Login{UserID: users[len(users)-1].ID}
	suite.Require().Nil(suite.db.Create(&login))

	actualUser := new(User)

	// Act
	err = suite.db.Joins("JOIN logins ON logins.user_id = users.id").First(actualUser)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(users[len(users)-1].ID, actualUser.ID)

	// Arrange
	now := time.Now()
	logins := []*Login{
		{At: now.AddDate(0, 0, -1), UserID: users[0].ID},
		{At: now.AddDate(0, 0, -2), UserID: users[1].ID},
		{At: now.AddDate(0, 0, -3), UserID: users[2].ID},
	}
	suite.Require().Nil(suite.db.Create(&logins))

	var actualUsers []User

	// Act
	err = suite.db.Joins("JOIN logins ON logins.user_id = users.id AND logins.at::date >= ?::date", now.AddDate(0, 0, -1)).
		Find(&actualUsers)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualUsers, 1)

	// Arrange
	suite.Require().Nil(suite.db.Create(&Login{UserID: users[0].ID}))

	subQ := suite.db.Model(new(User)).Where("account_id = ?", accts[0].ID).Select("id")

	var actualLogins []Login

	// Act
	err = suite.db.Joins("JOIN (?) AS users ON users.id = logins.user_id", subQ).
		Find(&actualLogins)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualLogins, 2)

	// Arrange
	groups := insertGroups(suite.T(), suite.db)

	var groupUser []GroupUser
	for _, user := range users {
		groupUser = append(groupUser, GroupUser{GroupID: groups[0].ID, UserID: user.ID})
	}
	suite.Require().Nil(suite.db.Create(&groupUser))

	var actualGroup []Group

	// Act
	err = suite.db.Joins("JOIN group_users ON group_users.group_id = groups.id").
		Joins("JOIN users ON users.id = group_users.user_id").
		Find(&actualGroup)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualGroup, len(users))
	for _, group := range actualGroup {
		suite.Require().Equal(groups[0].ID, group.ID)
	}
}

func (suite *DBTestSuite) TestLimit() {
	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	var limit int
	var actual []Account

	// Act
	err := suite.db.Limit(limit).Find(&actual)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotFound)
	suite.Require().Len(actual, 0)

	// Arrange
	limit = 2

	// Act
	err = suite.db.Limit(limit).Find(&actual)
	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 2)

	// Arrange
	limit = -1
	actual = []Account{}

	// Act
	err = suite.db.Limit(limit).Find(&actual)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)
	suite.Require().Len(actual, 0)
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
	accts := insertAccounts(suite.T(), suite.db)

	var offset int
	actual := new(Account)

	// Act
	err := suite.db.Offset(offset).First(actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[offset].ID, actual.ID)

	// Arrange
	offset = 3
	actual = new(Account)

	// Act
	err = suite.db.Offset(offset).First(actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[offset].ID, actual.ID)

	// Arrange
	offset = -2
	actual = new(Account)

	// Act
	err = suite.db.Offset(offset).First(actual)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)
}

func (suite *DBTestSuite) TestOr() {
	// Arrange
	_ = insertAccounts(suite.T(), suite.db)

	var actual []Account

	// Act
	err := suite.db.Or("kind = ?", "exceptional").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 2)

	// Arrange
	actual = []Account{}

	// Act
	err = suite.db.Or("kind = ?", "default").Or("kind = ?", "exceptional").Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 4)

	// Arrange
	actual = []Account{}

	badSubQ := suite.db.Where("kind = ?", "exceptional").Select("id")

	// Act
	err = suite.db.Where("kind = ?", "default").Or("id IN (?)", badSubQ).Find(&actual)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	actual = []Account{}

	goodSubQ := suite.db.Or("kind = ?", "default").Or("kind = ?", "exceptional")

	// Act
	err = suite.db.Where(goodSubQ).Find(&actual)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actual, 4)
}

func (suite *DBTestSuite) TestOrder() {
	// Arrange
	groups := insertGroups(suite.T(), suite.db)

	var actualGroups []Group

	// Act
	err := suite.db.Order("").Find(&actualGroups)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualGroups, len(groups))
	for i := range actualGroups {
		suite.Require().Equal(groups[i].ID, actualGroups[i].ID)
	}

	// Arrange
	actualGroups = []Group{}

	// Act
	err = suite.db.Order("starts DESC").Find(&actualGroups)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualGroups, len(groups))
	for i := range actualGroups {
		suite.Require().Equal(groups[len(groups)-1-i].ID, actualGroups[i].ID)
	}

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	users := insertUsers(suite.T(), suite.db, accts)

	var actualUsers []User

	// Act
	err = suite.db.Order("status ASC, role DESC").Find(&actualUsers)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualUsers, len(users))
	suite.Require().Equal("inactive", actualUsers[len(actualUsers)-1].Status)
	suite.Require().Equal("manager", actualUsers[len(actualUsers)-1].Role)
	suite.Require().Equal("inactive", actualUsers[len(actualUsers)-2].Status)
	suite.Require().Equal("owner", actualUsers[len(actualUsers)-2].Role)
}

func (suite *DBTestSuite) TestPaged() {
	// Arrange
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	actual, err := db.Paged(0, 0)

	// Assert
	suite.Require().ErrorIs(err, testErr)
	suite.Require().Zero(actual)

	// Arrange + Act
	actual, err = suite.db.Paged(1, 1)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnaddressable)

	// Arrange
	notModel := "hello"

	// Act
	actual, err = suite.db.Model(notModel).Paged(1, 1)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)

	// Arrange + Act
	actual, err = suite.db.Model(new(User)).Paged(1, 1)

	// Assert
	suite.Require().Nil(err)
	suite.Require().NotNil(actual.Items)
	v, ok := actual.Items.(*[]User)
	suite.Require().True(ok)
	suite.Require().Len(*v, 0)
	suite.Require().Equal(int64(1), actual.Page)
	suite.Require().Equal(int64(1), actual.PerPage)
	suite.Require().Equal(int64(0), actual.TotalItems)
	suite.Require().Equal(int64(0), actual.TotalPages)

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	users := insertUsers(suite.T(), suite.db, accts)

	// Act
	actual, err = suite.db.Model(new(User)).Paged(1, 2)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(int64(1), actual.Page)
	suite.Require().Equal(int64(2), actual.PerPage)
	suite.Require().Equal(int64(len(users)), actual.TotalItems)
	suite.Require().Equal(int64(5), actual.TotalPages)

	v, ok = actual.Items.(*[]User)
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
		{At: time.Now(), UserID: expectedU.ID},
		{At: time.Now(), UserID: expectedU.ID},
		{At: time.Now(), UserID: expectedU.ID},
		{At: time.Now(), UserID: expectedU.ID},
		{At: time.Now(), UserID: expectedU.ID},
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
	// Arrange
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	err := db.Raw(nil, "SELECT * FROM accounts")

	// Assert
	suite.Require().ErrorIs(err, testErr)

	// Arrange
	q := "not a real statement"
	var actual Account

	// Act
	err = suite.db.Raw(&actual, q)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	var notAccount string

	// Act
	err = suite.db.Raw(&notAccount, "SELECT * FROM accounts")

	// Assert
	suite.Require().Nil(err)
	suite.Require().Zero(notAccount)

	// Arrange
	accts := insertAccounts(suite.T(), suite.db)
	q = "SELECT id, kind FROM accounts WHERE id = ?;"

	// Act
	err = suite.db.Raw(&actual, q, accts[0].ID)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Equal(accts[0].ID, actual.ID)
	suite.Require().Equal(accts[0].Kind, actual.Kind)

	// Arrange
	q = "SELECT id, kind FROM accounts WHERE id = 1;"

	// Act
	err = suite.db.Raw(nil, q)

	// Assert
	suite.Require().Nil(err)

	// Arrange
	notAPointer := "not a pointer"

	// Act
	err = suite.db.Raw(notAPointer, "SELECT kind FROM accounts LIMIT 1")

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnaddressable)
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
	// Arrange
	accts := insertAccounts(suite.T(), suite.db)

	var actualAccounts []Account

	// Act
	err := suite.db.Where("kind = ?", "exceptional").Find(&actualAccounts)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualAccounts, 2)

	// Arrange
	_ = insertUsers(suite.T(), suite.db, accts)

	var actualUsers []User

	// Act
	err = suite.db.Where("users.account_id = ? AND users.role = ?", accts[0].ID, "owner").Find(&actualUsers)

	// Arrange
	actualUsers = []User{}

	badSubq := suite.db.Where("kind = ?", "special").Select("id")

	// Act
	err = suite.db.Where("account_id IN (?)", badSubq).Find(&actualUsers)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotValid)

	// Arrange
	goodSubq := suite.db.Model(new(Account)).Where("kind = ?", "special").Select("id")

	// Act
	err = suite.db.Where("account_id IN (?)", goodSubq).Find(&actualUsers)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualUsers, 2)

	// Arrange
	actualUsers = []User{}

	filter := suite.db.Where("role = ?", "owner")

	// Act
	err = suite.db.Where(filter).Find(&actualUsers)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualUsers, 5)

	// Arrange
	groups := insertGroups(suite.T(), suite.db)

	updates := postgres.Updates{"null_string": "Hello, World!"}
	suite.Require().Nil(suite.db.Model(new(Group)).Where("id != ?", groups[0].ID).Update(updates))

	var actualGroups []Group

	// Act
	err = suite.db.Where("null_string", nil).Find(&actualGroups)

	// Assert
	suite.Require().Nil(err)
	suite.Require().Len(actualGroups, 1)
	suite.Require().Equal(actualGroups[0], groups[0])
}

func (suite *DBTestSuite) TestRollback() {
	// Arrange
	tx := suite.db.Begin()
	acct := Account{Kind: "rollback-test"}
	suite.Require().Nil(tx.Create(&acct))
	suite.Require().NotZero(acct.ID)

	// Act
	err := tx.Rollback()

	// Assert
	suite.Require().Nil(err)

	var actual Account
	suite.Require().ErrorIs(suite.db.Where("id = ?", acct.ID).First(&actual), trails.ErrNotFound)
}

func (suite *DBTestSuite) TestUpdate() {
	// Arrange
	db := postgres.NewDB(suite.db.DB().Session(&gorm.Session{NewDB: true}))
	db.DB().Error = testErr

	// Act
	err := db.Update(nil)

	// Assert
	suite.Require().ErrorIs(err, testErr)

	// Arrange
	updates := make(postgres.Updates)

	// Act
	err = suite.db.Model(new(User)).Where("id = ?", 2).Update(updates)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrMissingData)

	// Arrange
	updates["name"] = "Jimothy Bobbitz"

	// Act
	err = suite.db.Debug().Model(new(User)).Where("id = ?", 2).Update(updates)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrNotFound)

	// Arrange
	acct := new(Account)
	suite.Require().Nil(suite.db.Create(acct))

	user := &User{AccountID: acct.ID, Name: "Jimothy Bobbitz"}
	suite.Require().Nil(suite.db.Create(user))

	// Act
	err = suite.db.Model(new(User)).Where("id = ?", user.ID).Update(updates)

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

	// Arrange
	updates = postgres.Updates{"fake-column": "fake-value"}

	// Act
	err = suite.db.Model(new(User)).Where("id = ?", user.ID).Update(updates)

	// Assert
	suite.Require().ErrorIs(err, trails.ErrUnexpected)
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
