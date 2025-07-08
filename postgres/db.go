package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/xy-planning-network/trails"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

// TODO(dlk): docstring
func (db *DB) DB() *gorm.DB { return db.db }

// TODO(dlk): docstring
func NewDB(db *gorm.DB) *DB { return &DB{db: db} }

// TODO(dlk): docstring
func (db *DB) Begin(opts ...*sql.TxOptions) *DB {
	return &DB{db: db.db.Begin(opts...)}
}

// TODO(dlk): docstring
func (db *DB) Count() (int64, error) {
	var count int64
	if err := db.db.Count(&count).Error; err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return 0, err
	}

	return count, nil
}

// TODO(dlk): docstring
func (db *DB) Commit() error {
	if err := db.db.Commit().Error; err != nil {
		err = fmt.Errorf("%w: failed committing tx: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

// Create inserts value into the database, updating value with new data yielding from that insertion.
//
// Value must be a pointer, otherwise ErrUnaddressable returns.
// If value violates a foreign key constraint defined by the database, ErrNotValid returns.
// If value violates a unique constraint defined by the database, ErrExists returns.
func (db *DB) Create(value any) error {
	// NOTE(dlk): GORM panics in either of these cases, let's mitigate those.
	if value == nil {
		return fmt.Errorf("%w: value cannot be nil", trails.ErrUnaddressable)
	}

	if reflect.TypeOf(value).Kind() != reflect.Pointer {
		return fmt.Errorf("%w: %T must be a pointer", trails.ErrUnaddressable, value)
	}

	err := db.db.Create(value).Error
	switch {
	case err == nil:
		return nil

	case strings.Contains(err.Error(), violatesFK):
		return fmt.Errorf("%w: %s", trails.ErrNotValid, err)

	case strings.Contains(err.Error(), violatesUniq):
		return fmt.Errorf("%w: %s", trails.ErrExists, err)

	default:
		return fmt.Errorf("%w: failed creating %T: %s", trails.ErrUnexpected, value, err)
	}
}

// TODO(dlk): docstring
func (db *DB) Delete(value any) error {
	if err := db.db.Delete(value).Error; err != nil {
		err = fmt.Errorf("%w: failed deleting %T: %s", trails.ErrUnexpected, value, err)
		return err
	}
	return nil
}

// TODO(dlk): docstring
func (db *DB) Debug() *DB { return &DB{db.db.Debug()} }

// TODO(dlk): docstring
func (db *DB) Distinct(args ...any) *DB { return &DB{db.db.Distinct(args...)} }

// TODO(dlk): docstring
func (db *DB) Exec(sql string, values ...any) error {
	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}

	if res.RowsAffected == 0 { // TODO(dlk):
		return fmt.Errorf("%w: exec failed to affect any rows", trails.ErrNotFound)
	}

	return nil
}

// TODO(dlk): docstring
func (db *DB) Exists() (bool, error) {
	var exists bool
	err := db.db.Raw("SELECT exists(?)", nil).Error // TODO
	if err != nil {
		// TODO err wrap
		return exists, err
	}

	return exists, nil
}

// TODO(dlk): docstring
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

// TODO(dlk): docstring
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

// TODO(dlk): docstring
func (db *DB) Group(name string) *DB { return &DB{db: db.db.Group(name)} }

// TODO(dlk): docstring
func (db *DB) Joins(query string, args ...any) *DB {
	return &DB{db: db.db.Joins(query, args...)}
}

// TODO(dlk): docstring
func (db *DB) Limit(limit int) *DB { return &DB{db: db.db.Limit(limit)} }

// Model declares the table used for the query.
//
// Model computes the name for the database table from the type of model,
// taking the plural of the table, for example:
// - Account -> accounts
// - User -> users
//
// Unless, model implements: func TableName() string
// The value returned from that function is used instead.
//
// Calling Model multiple times is undefined behavior.
// Experience indicates the last call in the chain sets the table - buyer beware.
func (db *DB) Model(model any) *DB { return &DB{db: db.db.Model(model)} }

// TODO(dlk): docstring
func (db *DB) Offset(offset int) *DB { return &DB{db: db.db.Offset(offset)} }

// TODO(dlk): docstring
func (db *DB) Or(query string, args ...any) *DB {
	return &DB{db: db.db.Or(query, args...)}
}

// TODO(dlk): docstring
func (db *DB) Order(order string) *DB { return &DB{db: db.db.Order(order)} }

// TODO(dlk): docstring
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

// Preload fetches data embedded in a model based on that model's associations.
// An association can be specified by the model's field name, such as Account or User.
//
// To load all associations, use the constant Associations.
//
// Nested preloading is also possible by using dot syntax: User.GroupUser.Group.
//
// If additional filtering on the preloaded data is necessary,
// scopes can be used to add conditions:
//
//	adminScope := func(dbx *DB) *DB { return dbx.Where("role = ?", "admin") }
//	db.Preload("User", adminScope).Where("id = ?", id).First(&account)
func (db *DB) Preload(association string, scopes ...Scope) *DB {
	var resolved []any
	dbx := NewDB(db.DB())
	for _, scope := range scopes {
		resolved = append(resolved, scope(dbx).DB())
	}

	return &DB{db: db.db.Preload(association, resolved...)}
}

// TODO(dlk): docstring
func (db *DB) Raw(dest any, sql string, values ...any) error {
	err := db.db.Raw(sql, values...).Scan(dest).Error
	if err != nil {
		err = fmt.Errorf("%w: failed scanning results: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

// Scope applies the scope to the existing query.
// Review [Scope] for more details.
func (db *DB) Scope(scope Scope) *DB {
	return &DB{db: db.db.Scopes(func(dbx *gorm.DB) *gorm.DB {
		return scope(NewDB(dbx)).DB()
	})}
}

// TODO(dlk): docstring
func (db *DB) Select(columns ...string) *DB { return &DB{db: db.db.Select(columns)} }

// TODO(dlk): docstring
func (db *DB) Table(name string) *DB { return &DB{db: db.db.Table(name)} }

// TODO(dlk): docstring
func (db *DB) Unscoped() *DB { return &DB{db: db.db.Unscoped()} }

// TODO(dlk): docstring
func (db *DB) Where(query string, args ...any) *DB {
	return &DB{db.db.Where(query, args...)}
}

// TODO(dlk): docstring
func (db *DB) Rollback() error {
	err := db.db.Rollback().Error
	if err != nil {
		return fmt.Errorf("%w: failed rolling back tx: %s", trails.ErrUnexpected, err)
	}

	return nil
}

// TODO(dlk): docstring
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
