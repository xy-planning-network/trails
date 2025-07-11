package postgres_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/postgres"
)

type DBTestSuite struct {
	suite.Suite

	db *postgres.DB
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}

var cfg = postgres.Config{
	Env:      trails.Testing,
	IsTestDB: true,
	Host:     "localhost",
	Port:     "5432",
	Name:     "trails_test",
	User:     "trails_test",
	Schema:   "public",
	Password: "c7b4eb410a5d70ee6976a3a7dd57937c",
}

func (suite *DBTestSuite) SetupSuite() {
	var err error
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
