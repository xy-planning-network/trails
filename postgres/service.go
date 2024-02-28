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

type Simple[T any] struct {
	db *gorm.DB
}

func NewSimple[T any](db *gorm.DB) *Simple[T] {
	return &Simple[T]{db}
}

func (db *Simple[T]) Debug() *Simple[T] {
	return &Simple[T]{db.db.Debug()}
}

func (db *Simple[T]) Distinct(args ...interface{}) *Simple[T] {
	return &Simple[T]{db.db.Distinct(args...)}
}

func (db *Simple[T]) Find() ([]T, error) {
	dest := make([]T, 0)
	if err := db.db.Model(new(T)).Find(&dest).Error; err != nil {
		return dest, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return dest, nil
}

func (db *Simple[T]) First() (T, error) {
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

func (db *Simple[T]) Group(name string) *Simple[T] { return db }

func (db *Simple[T]) Joins(query string, args ...interface{}) *Simple[T] { return db }

func (db *Simple[T]) Limit(limit int) *Simple[T] { return db }

func (db *Simple[T]) Offset(offset int) *Simple[T] { return db }

func (db *Simple[T]) Or(query interface{}, args ...interface{}) *Simple[T] { return db }

func (db *Simple[T]) Order(value interface{}) *Simple[T] { return db }

func (db *Simple[T]) Paged(page, perPage int) (PagedData[T], error) {
	var pd PagedData[T]
	var items T

	// Make sure page/perPage are sane
	page = max(1, page)
	perPage = max(1, perPage)

	// Conduct unlimited count query to calculate totals
	var totalRecords int64
	if err := db.db.Model(&items).Session(new(gorm.Session)).Count(&totalRecords).Error; err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return pd, err
	}

	// Calculate offset and conduct limited query
	offset := (page - 1) * perPage
	if err := db.db.Model(&items).Limit(perPage).Offset(offset).Find(&items).Error; err != nil {
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

func (db *Simple[T]) Preload(query string, args ...interface{}) *Simple[T] { return db }

func (db *Simple[T]) Scopes(funcs ...func(*Simple[T]) *Simple[T]) *Simple[T] { return db }

func (db *Simple[T]) Select(columns ...string) *Simple[T] { return db }

// Table specifies the name of the table to query when T is not a struct matching a database table.
//
// e.g.:
//
//	ids, err := NewSimple[[]uint].Table("users").Select("id").Find()
func (db *Simple[T]) Table(name string) *Simple[T] { return db }

func (db *Simple[T]) Unscoped() *Simple[T] { return db }

func (db *Simple[T]) Where(query interface{}, args ...interface{}) *Simple[T] {
	return &Simple[T]{db.db.Where(query, args...)}
}

type Robust[T any] struct {
	*Simple[T]
}

func (db *Robust[T]) Begin(opts ...*sql.TxOptions) *Robust[T] { return db }

func (db *Robust[T]) Count() (dest T, err error) { return dest, err }

func (db *Robust[T]) Commit() error { return nil }

func (db *Robust[T]) Create(value T) error { return nil }

// escape hatch
func (db *Robust[T]) DB() *gorm.DB { return db.db }

func (db *Robust[T]) Debug() *Robust[T] {
	return &Robust[T]{db.Simple.Debug()}
}

func (db *Robust[T]) Delete(value T) error { return nil }

func (db *Robust[T]) Distinct(args ...interface{}) *Robust[T] {
	return &Robust[T]{db.Simple.Distinct(args...)}
}

func (db *Robust[T]) Group(name string) *Robust[T] { return db }

func (db *Robust[T]) Joins(query string, args ...interface{}) *Robust[T] { return db }

func (db *Robust[T]) Limit(limit int) *Robust[T] { return db }

func (db *Robust[T]) Offset(offset int) *Robust[T] { return db }

func (db *Robust[T]) Or(query interface{}, args ...interface{}) *Robust[T] { return db }

func (db *Robust[T]) Order(value interface{}) *Robust[T] { return db }

func (db *Robust[T]) Preload(query string, args ...interface{}) *Robust[T] { return db }

func (db *Robust[T]) Scopes(funcs ...func(*Robust[T]) *Robust[T]) *Robust[T] { return db }

func (db *Robust[T]) Select(columns ...string) *Robust[T] { return db }

func (db *Robust[T]) Table(name string) *Robust[T] { return db }

func (db *Robust[T]) Unscoped() *Robust[T] { return db }

func (db *Robust[T]) Where(query interface{}, args ...interface{}) *Robust[T] {
	return &Robust[T]{db.Simple.Where(query, args)}
}

func (db *Robust[T]) Exec(sql string, values ...interface{}) (int64, error) {
	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		return 0, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return res.RowsAffected, nil
}

func (db *Robust[T]) Raw(sql string, values ...interface{}) (dest T, err error) { return dest, err }

func (db *Robust[T]) Rollback() *Robust[T] { return db }

func (db *Robust[T]) Update(column string, value interface{}) (dest []T, err error) { return dest, err }

func (db *Robust[T]) Updates(values map[string]interface{}) ([]T, error) {
	dest := make([]T, 0)
	err := db.db.Model(new(T)).Clauses(clause.Returning{} /* GORM's RETURNING * syntax */).Updates(values).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return nil, err
	}

	return dest, nil
}

// PagedData is returned from the Paged method. It contains paged database records and pagination metadata.
type PagedData[T any] struct {
	Items      T     `json:"items"`
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}
