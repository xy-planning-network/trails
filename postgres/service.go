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

func (db *Simple[T]) Debug() *Simple[T] { return db }

func (db *Simple[T]) Distinct(args ...interface{}) *Simple[T] { return db }

func (db *Simple[T]) Find() (T, error) { return nil }

func (db *Simple[T]) First() (T, error) {
	var dest T
	err := db.db.Model(&dest).First(&dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	if err != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return nil
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
func (db *Simple[T]) Table(name string) *Simple[T] {}

func (db *Simple[T]) Unscoped() *Simple[T] { return db }

func (db *Simple[T]) Where(query interface{}, args ...interface{}) *Simple[T] {
	return &Simple[T]{db.db.Where(query, args...)}
}

type Robust[T any] struct {
	*Simple[T]
}

func (db *Robust[T]) Begin(opts ...*sql.TxOptions) *Robust[T] { return db }

func (db *Robust[T]) Commit() error { return nil }

func (db *Robust[T]) Create(value T) error { return nil }

// escape hatch
func (db *Robust[T]) DB() *gorm.DB { return db.db }

func (db *Robust[T]) Delete(value T) error { return nil }

func (db *Robust[T]) Exec(sql string, values ...interface{}) (int64, error) {
	var err error
	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}
	return res.RowsAffected, err
}

func (db *Robust[T]) Raw(dest interface{}, sql string, values ...interface{}) error { return nil }

func (db *Robust[T]) Rollback() *Robust[T] { return db }

func (db *Robust[T]) Update(column string, value interface{}) (T, error) { return db }

func (db *Robust[T]) Updates(values interface{}) (T, error) {
	var dest T
	err := db.db.Model(&dest).Clauses(clause.Returning{} /* GORM's RETURNING * syntax */).Updates(values).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return dest, err
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

// DatabaseServiceImpl satisfies the above DatabaseService interface.
type DatabaseServiceImpl struct {
	DB *gorm.DB
}

// NewService hydrates the gorm database for the implementation struct methods.
func NewService(DB *gorm.DB) *DatabaseServiceImpl {
	return &DatabaseServiceImpl{DB: DB}
}

// CountByQuery recives a database model and query and fetches a count for the given params.
func (service *DatabaseServiceImpl) CountByQuery(model any, query map[string]any) (int64, error) {
	count := int64(0)
	return count, service.DB.Model(model).Where(query).Count(&count).Error
}

// FetchByQuery receives a slice of database models as a pointer and fetches all records matching the query.
func (service *DatabaseServiceImpl) FetchByQuery(models any, query string, params []any) error {
	return service.DB.Where(query, params...).Find(models).Error
}

// FindByID receives a database model as a pointer and fetches it using the primary ID.
func (service *DatabaseServiceImpl) FindByID(model any, ID any) error {
	return service.DB.First(model, ID).Error
}

// FindByQuery receives a database model as a pointer and fetches it using the given query.
func (service *DatabaseServiceImpl) FindByQuery(model any, query map[string]any) error {
	return service.DB.Where(query).First(model).Error
}

// Insert receives a database model and inserts it into the database.
func (service *DatabaseServiceImpl) Insert(model any) error {
	return service.DB.Create(model).Error
}

// PagedByQuery receives a slice of database models and paging information to build a paged database query.
func (service *DatabaseServiceImpl) PagedByQuery(models any, query string, params []any, order string, page int, perPage int, preloads ...string) (PagedData, error) {
	pd := PagedData{}

	// Make sure page/perPage are sane
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Conduct unlimited count query to calculate totals
	var totalRecords int64
	if err := service.DB.Where(query, params...).Model(models).Count(&totalRecords).Error; err != nil {
		return pd, err
	}

	// Calculate offset and conduct limited query
	offset := (page - 1) * perPage
	session := service.DB
	for _, preload := range preloads {
		session = session.Preload(preload)
	}
	if err := session.Where(query, params...).Order(order).Limit(perPage).Offset(offset).Find(models).Error; err != nil {
		return pd, err
	}

	pd.Items = models
	pd.Page = page
	pd.PerPage = perPage
	pd.TotalItems = totalRecords
	totalPagesFloat := float64(totalRecords) / float64(perPage)
	pd.TotalPages = int(math.Ceil(totalPagesFloat))

	return pd, nil
}

// PagedByQueryFromSession receives a slice of database models and paging information to build a paged database query.
func (service *DatabaseServiceImpl) PagedByQueryFromSession(models any, session *gorm.DB, page int, perPage int) (PagedData, error) {
	pd := PagedData{}

	// Make sure page/perPage are sane
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Conduct unlimited count query to calculate totals
	var totalRecords int64
	if err := session.Model(models).Count(&totalRecords).Error; err != nil {
		return pd, err
	}

	// Calculate offset and conduct limited query
	offset := (page - 1) * perPage
	if err := session.Session(&gorm.Session{QueryFields: true}).Limit(perPage).Offset(offset).Find(models).Error; err != nil {
		return pd, err
	}

	pd.Items = models
	pd.Page = page
	pd.PerPage = perPage
	pd.TotalItems = totalRecords
	totalPagesFloat := float64(totalRecords) / float64(perPage)
	pd.TotalPages = int(math.Ceil(totalPagesFloat))

	return pd, nil
}
