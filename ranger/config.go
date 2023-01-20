package ranger

import (
	"errors"
	"fmt"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/http/middleware"
	"github.com/xy-planning-network/trails/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config[User RangerUser] struct {
	// NOTE(dlk): Ranger can accept a type parameter also.
	// Config was chosen to minimize proliferating generic type parameters
	// in all Ranger methods or references to Ranger.
	// Config ought to be restricted to New.
}

// defaultUserStore constructs a function matching the signature of middleware.UserStorer.
// This function pulls the User from the db by ID,
// preloading all top-level associations.
func (Config[User]) defaultUserStore(db postgres.DatabaseService) middleware.UserStorer {
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
		var user User
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
