package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/xy-planning-network/trails"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// None is useful as a type for DB[T any] when no model is involved in the query.
// Examples are in Exec queries where the query
type None struct{}

type DB[T any] struct {
	db *gorm.DB
}

func Query[T any]() *DB[T] { return &DB[T]{DefaultConn.db} }

func (db *DB[T]) Begin(opts ...*sql.TxOptions) *DB[T] {
	return &DB[T]{db: db.db.Begin(opts...)}
}

func (db *DB[T]) Count() (int64, error) {
	var count int64
	if err := db.db.Count(&count).Error; err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return 0, err
	}

	return count, nil
}

func (db *DB[T]) Commit() error {
	if err := db.db.Commit().Error; err != nil {
		err = fmt.Errorf("%w: failed committing tx: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

func (db *DB[T]) Create(value T) error {
	if err := db.db.Create(value).Error; err != nil {
		err = fmt.Errorf("%w: failed creating %T: %s", trails.ErrUnexpected, value, err)
		return err
	}

	return nil
}

func (db *DB[T]) Delete(value T) error { return nil } // TODO

func (db *DB[T]) Debug() *DB[T] { return &DB[T]{db.db.Debug()} }

func (db *DB[T]) Distinct(args ...interface{}) *DB[T] {
	return &DB[T]{db.db.Distinct(args...)}
}

func (db *DB[T]) Exec(sql string, values ...interface{}) (int64, error) {
	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		return 0, fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}

	return res.RowsAffected, nil
}

func (db *DB[T]) Exists() (bool, error) { // TODo
	var exists bool
	err := db.db.Raw("SELECT exists(?)", nil).Error
	if err != nil {
		// TODO err wrap
		return exists, err
	}

	return exists, nil
}

func (db *DB[T]) Find() ([]T, error) {
	var dest []T
	if err := db.db.Model(new(T)).Find(&dest).Error; err != nil {
		return dest, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	if len(dest) == 0 {
		return nil, fmt.Errorf("%w", trails.ErrNotFound)
	}

	return dest, nil
}

func (db *DB[T]) First() (T, error) {
	var dest T
	err := db.db.Model(&dest).First(&dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dest, fmt.Errorf("%w", trails.ErrNotFound)
	}

	if err != nil {
		return dest, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return dest, nil
}

func (db *DB[T]) Group(name string) *DB[T] { return db } // TODO

func (db *DB[T]) Joins(query string, args ...interface{}) *DB[T] { return db } // TODO

func (db *DB[T]) Limit(limit int) *DB[T] { return db } // TODO

func (db *DB[T]) Offset(offset int) *DB[T] { return db } // TODO

func (db *DB[T]) Or(query interface{}, args ...interface{}) *DB[T] { return db } // TODO

func (db *DB[T]) Order(value interface{}) *DB[T] {
	return &DB[T]{db.db.Order(value)}
}

func (db *DB[T]) Paged(page, perPage int) (PagedData[T], error) {
	var pd PagedData[T]
	var items T

	page = max(1, page)
	perPage = max(1, perPage)

	var totalRecords int64
	err := db.db.Model(&items).Session(new(gorm.Session)).Count(&totalRecords).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return pd, err
	}

	offset := (page - 1) * perPage
	err = db.db.Model(&items).Limit(perPage).Offset(offset).Find(&items).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return pd, err
	}

	pd.Items = items
	pd.Page = page
	pd.PerPage = perPage
	pd.TotalItems = totalRecords
	totalPagesFloat := float64(totalRecords) / float64(perPage)
	pd.TotalPages = int(math.Ceil(totalPagesFloat))

	return pd, nil
}

func (db *DB[T]) Preload(query string, args ...interface{}) *DB[T] {
	return &DB[T]{db: db.db.Preload(query, args)}
}

func (db *DB[T]) Scopes(funcs ...func(*DB[T]) *DB[T]) *DB[T] { return db }

func (db *DB[T]) Select(columns ...string) *DB[T] { return db }

// Table specifies the name of the table to query
// Use when T is not a struct matching a database table,
// or, T is the table targetted in the query, but not the type returned.
//
// For example:
//
//	   ids, err := Query[[]uint].Table("users").Select("id").Find()
//
//	   users, err := Query[[]User].
//		       Table("accounts").
//		       Select("users.*").
//		       Joins("JOIN users ON accounts.id = users.account_id").
//		       Where("accounts.active").
//		       Find()
//
// TODO(dlk):
// - [ ] implement
// - [ ] confirm Model usage elsewhere doesn't collide
func (db *DB[T]) Table(name string) *DB[T] { return db }

// TODO
func (db *DB[T]) Unscoped() *DB[T] { return db }

func (db *DB[T]) Where(query interface{}, args ...interface{}) *DB[T] {
	return &DB[T]{db.db.Where(query, args...)}
}

// TODO
func (db *DB[T]) Raw(sql string, values ...interface{}) (dest T, err error) { return dest, err }

func (db *DB[T]) Rollback() error {
	err := db.db.Rollback().Error
	if err != nil {
		return fmt.Errorf("%w: failed rolling back tx: %s", trails.ErrUnexpected, err)
	}

	return nil
}

func (db *DB[T]) Updates(values map[string]interface{}) ([]T, error) {
	var dest []T
	err := db.db.Model(&dest).Clauses(clause.Returning{}).Updates(values).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return nil, err
	}

	if len(dest) == 0 {
		return nil, fmt.Errorf("%w", trails.ErrNotFound)
	}

	return dest, nil
}

// PagedData is returned from the Paged method.
// It contains paged database records and pagination metadata.
type PagedData[T any] struct {
	Items      T     `json:"items"`
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}
