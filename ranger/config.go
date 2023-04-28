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
	"gorm.io/gorm/clause"
)

type Config[U RangerUser] struct {
	// NOTE(dlk): Ranger can accept a type parameter also, like how New does.
	// Config was chosen to minimize proliferating generic type parameters
	// in all Ranger methods or references to Ranger.
	// Config ought to be restricted to New.

	// FS is the filesystem to find templates in for rendering them.
	FS fs.FS

	// Migrations are a list of DB migrations to run upon DB successful connection.
	Migrations []postgres.Migration

	// Shutdowns are a series of functions that ought to be called before *Ranger
	// stops handling HTTP requests.
	Shutdowns []ShutdownFn

	mockdb    *postgres.MockDatabaseService
	logoutput io.Writer
}

// UseDBMock overrides a real database connection with a mocked database
// hooked up to ctrl.
func (c *Config[U]) UseDBMock(mockdb *postgres.MockDatabaseService) { c.mockdb = mockdb }

// UseLogOutput overrides the writing logs to os.Stdout;
// use a bytes.Buffer in unit tests so log outputs can be inspected.
func (c *Config[U]) UseLogOutput(w io.Writer) { c.logoutput = w }

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

	// NOTE(dlk): if ranger.Ranger.db was a *postgres.DatabaseServiceImpl
	// instead of *postgres.DatabaseService,
	// the type assertion would not be necessary;
	// we are not ready to commit to this inflexibilty, yet.
	if db, ok := db.(*postgres.DatabaseServiceImpl); ok {
		findByID = func(model, id any) error {
			return db.DB.Preload(clause.Associations).First(model, id).Error
		}
	}

	return func(id uint) (middleware.User, error) {
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

type WorkerConfig struct {
	// Migrations are a list of DB migrations to run upon DB successful connection.
	Migrations []postgres.Migration
}
