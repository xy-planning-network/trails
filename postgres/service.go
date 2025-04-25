package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/xy-planning-network/trails"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func (db *DB) Begin(opts ...*sql.TxOptions) *DB {
	return &DB{db: db.db.Begin(opts...)}
}

func (db *DB) Count() (int64, error) {
	var count int64
	if err := db.db.Count(&count).Error; err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return 0, err
	}

	return count, nil
}

func (db *DB) Commit() error {
	if err := db.db.Commit().Error; err != nil {
		err = fmt.Errorf("%w: failed committing tx: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

func (db *DB) Create(value any) error {
	if err := db.db.Create(value).Error; err != nil {
		err = fmt.Errorf("%w: failed creating %T: %s", trails.ErrUnexpected, value, err)
		return err
	}

	return nil
}

func (db *DB) Delete(value any) error { return nil } // TODO

func (db *DB) Debug() *DB { return &DB{db.db.Debug()} }

func (db *DB) Distinct(args ...any) *DB {
	return &DB{db.db.Distinct(args...)}
}

func (db *DB) Exec(sql string, values ...any) error {
	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: exec failed to affect any rows", trails.ErrNotFound)
	}

	return nil
}

func (db *DB) Exists() (bool, error) { // TODO
	var exists bool
	err := db.db.Raw("SELECT exists(?)", nil).Error
	if err != nil {
		// TODO err wrap
		return exists, err
	}

	return exists, nil
}

func (db *DB) Find(dest any) error {
	res := db.db.Find(dest)
	if res.Error != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	return nil
}

func (db *DB) First(dest any) error {
	err := db.db.First(dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	if err != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return nil
}

func (db *DB) Group(name string) *DB { return &DB{db: db.db.Group(name)} }

func (db *DB) Joins(query string, args ...any) *DB {
	return &DB{db: db.db.Joins(query, args...)}
}

func (db *DB) Limit(limit int) *DB { return &DB{db: db.db.Limit(limit)} }

func (db *DB) Model(model any) *DB { return &DB{db: db.db.Model(model)} }

func (db *DB) Offset(offset int) *DB { return &DB{db: db.db.Offset(offset)} }

func (db *DB) Or(query string, args ...any) *DB {
	return &DB{db: db.db.Or(query, args...)}
}

func (db *DB) Order(order string) *DB { return &DB{db: db.db.Order(order)} }

func (db *DB) Paged(page, perPage int) (PagedData, error) {
	pd := PagedData{
		Page:    max(1, page),
		PerPage: max(1, perPage),
	}

	var totalRecords int64
	err := db.db.Session(new(gorm.Session)).Count(&pd.TotalItems).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return pd, err
	}

	offset := (pd.Page - 1) * pd.PerPage
	err = db.db.Limit(pd.PerPage).Offset(offset).Find(&pd.Items).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return pd, err
	}

	totalPagesFloat := float64(totalRecords) / float64(perPage)
	pd.TotalPages = int(math.Ceil(totalPagesFloat))

	return pd, nil
}

func (db *DB) Preload(query string, args ...any) *DB {
	return &DB{db: db.db.Preload(query, args)}
}

func (db *DB) Raw(dest any, sql string, values ...any) error {
	err := db.db.Raw(sql, values...).Scan(dest).Error
	if err != nil {
		err = fmt.Errorf("%w: failed scanning results: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

func (db *DB) Scope(scope Scope) *DB { return &DB{db: db.db.Scopes(scope)} }

func (db *DB) Select(columns ...string) *DB { return &DB{db: db.db.Select(columns)} }

func (db *DB) Table(name string) *DB { return &DB{db: db.db.Table(name)} }

func (db *DB) Unscoped() *DB { return &DB{db: db.db.Unscoped()} }

func (db *DB) Where(query string, args ...any) *DB {
	return &DB{db.db.Where(query, args...)}
}

func (db *DB) Rollback() error {
	err := db.db.Rollback().Error
	if err != nil {
		return fmt.Errorf("%w: failed rolling back tx: %s", trails.ErrUnexpected, err)
	}

	return nil
}

func (db *DB) Update(values map[string]any) error {
	res := db.db.Updates(values)
	if res.Error != nil {
		err := fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
		return err
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	return nil
}

// PagedData is returned from the Paged method.
// It contains paged database records and pagination metadata.
type PagedData struct {
	Items      any   `json:"items"`
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}
