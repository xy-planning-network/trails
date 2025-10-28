package postgres_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/postgres"
	"github.com/xy-planning-network/trails/ranger"
)

var testErr = errors.New("just testing")

type DBTestSuite struct {
	suite.Suite

	db *postgres.DB
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}

func (suite *DBTestSuite) SetupSuite() {
	err := godotenv.Load("../.env")
	var pe *fs.PathError
	if err != nil && !errors.As(err, &pe) {
		suite.Require().FailNow(err.Error())
	}

	cfg := ranger.NewPostgresConfig(trails.Testing)

	suite.db, err = postgres.Connect(cfg)
	suite.Require().Nil(err)

	b, err := os.ReadFile("testdata/schema.sql")
	suite.Require().Nil(err)

	err = suite.db.Exec(string(b))
	suite.Require().ErrorIs(err, trails.ErrNotFound)
}

func (suite *DBTestSuite) TearDownTest() {
	suite.Require().Nil(postgres.WipeDB(suite.db.DB(), "public"))
}
