package ranger

import (
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/postgres"
	"gorm.io/gorm"
)

type Config[U RangerUser] struct {
	// NOTE(dlk): Ranger can accept a type parameter also, like how New does.
	// Config was chosen to minimize proliferating generic type parameters
	// in all Ranger methods or references to Ranger.
	// Config ought to be restricted to New.

	// FS is the filesystem to find templates in for rendering them.
	FS fs.FS

	// MaintMode determines how to configure ranger on ranger.New.
	// If true, it skips setting up a database connection and routes to a maintenance page.
	MaintMode bool

	// Migrations are a list of DB migrations to run upon DB successful connection.
	Migrations []postgres.Migration

	mockdb    *postgres.MockDatabaseService
	logoutput io.Writer
}

// UseLogOutput overrides the writing logs to os.Stdout;
// use a bytes.Buffer in unit tests so log outputs can be inspected.
func (c *Config[U]) UseLogOutput(w io.Writer) { c.logoutput = w }

// UseDBMock overrides a real database connection with a mocked database
// hooked up to ctrl.
//
// Deprecated
func (c *Config[U]) UseDBMock(mockdb *postgres.MockDatabaseService) { c.mockdb = mockdb }

// Valid asserts the Config has all required data,
// returning trails.ErrBadConfig if not.
func (c Config[U]) Valid() error {
	if c.FS == nil {
		return fmt.Errorf("%w: c.FS cannot be nil", trails.ErrBadConfig)
	}

	return nil
}

// defaultUserStore constructs a function matching the signature of middleware.UserStorer.
// This function pulls the User from the db by ID,
// preloading all top-level associations.
func (Config[U]) defaultUserStore(db postgres.DatabaseService) middleware.UserStorer {
	findByID := db.FindByID

	// FIXME(dlk): After fully deleting *postgres.DatabaseService,
	// type assertion is not necessary.
	if db, ok := db.(*postgres.DB); ok {
		findByID = func(model, id any) error {
			return db.Preload(postgres.Associations).Where("id = ?", id).First(model)
		}
	}

	return func(id int64) (middleware.User, error) {
		var user U
		err := findByID(&user, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = fmt.Errorf("%w: User %d", trails.ErrNotExist, id)
		}

		if err != nil {
			return nil, err
		}

		return user, nil
	}
}
