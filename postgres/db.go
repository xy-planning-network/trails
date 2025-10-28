package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/xy-planning-network/trails"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DB struct {
	// *gorm.DB's methods are generally unsafe to use.
	// Specifically, some *gorm.DB methods are not thread-safe
	// and mutate the state of the *gorm.DB backing DB.
	//
	// If a *gorm.DB method calls *gorm.DB.getInstance,
	// this appears to render a method "safe" since it creates a new pointer.
	//
	// If a *gorm.DB method does not, be aware.
	// One solution is to use *gorm.DB.Session to force a clean pointer.
	db *gorm.DB
}

// NewDB constructs a *DB from a *gorm.DB.
func NewDB(db *gorm.DB) *DB { return &DB{db: db} }

// DB exposes the underlying *gorm.DB backing DB.
//
// NB: use in exceptional circumstances only.
func (db *DB) DB() *gorm.DB { return db.db }

// Debug prints the current query to the logger.
func (db *DB) Debug() *DB { return &DB{db.db.Debug()} }

// **************************************************************************
// FINISHER METHODS
//
// These methods close out a current query, executing it.
// All finisher methods are terminal and cannot be chained.
// They return any errors occuring within the query chain
// or when executing the query.
// Unless returning a value, like Count does a number or Paged does PagedData,
// finisher methods expect a pointer data from the query can be inserted into.
// There are exceptions to this general principle.
//
// **************************************************************************

// Count returns the number of records matching the current query or an error.
func (db *DB) Count() (int64, error) {
	if db.db.Error != nil {
		return 0, db.db.Error
	}

	var count int64
	if err := db.db.Count(&count).Error; err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return 0, err
	}

	return count, nil
}

// Create inserts value into the database, updating value with new data yielding from that insertion.
// Accordinglt, almost always, value is a pointer to a struct that is a database table.
//
// Create allows for setting the table via Table or Model, as well.
// Value can be a map[string]any or an Updates, in this use case.
// With the latter type, certain procedures benefit starting with *DB.Update
// and switching to *DB.Create without having to construct a pointer to a struct
// and mapping all values in Updates to that struct.
//
// For example, this is valid:
//
//		err := db.Model(new(User)).Where("account_id = ?", accountID).Update(updates)
//	 	if errors.Is(err, ErrNotFound) {
//	   		updates["account_id"] = accountID
//			err = db.Model(new(User)).Create(updates)
//	 	}
//
// Value must be a pointer, otherwise ErrUnaddressable returns.
// If value violates a foreign key constraint defined by the database, ErrNotValid returns.
// If value violates a unique constraint defined by the database, ErrExists returns.
// If value does not implement gorm.TableNamer -
// that is, is not a database table -
// then, ErrMissingData returns.
func (db *DB) Create(value any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %T must be a non-nil pointer or slice", trails.ErrUnaddressable, value)
		}

		return
	}()

	if db.db.Error != nil {
		return db.db.Error
	}

	if v, ok := value.(Updates); ok {
		if err = v.valid(); err != nil {
			return err
		}

		value = map[string]any(v)
	}

	err = db.db.Session(&gorm.Session{FullSaveAssociations: false}).Create(value).Error
	switch {
	case err == nil:
		return nil

	case errors.Is(err, schema.ErrUnsupportedDataType), errors.Is(err, gorm.ErrInvalidData):
		return fmt.Errorf("%w: %T does not implement gorm.TableNamer", trails.ErrMissingData, value)

	case strings.Contains(err.Error(), violatesFK):
		return fmt.Errorf("%w: %s", trails.ErrNotValid, err)

	case errUniqViolation.MatchString(err.Error()):
		return fmt.Errorf("%w: %s", trails.ErrExists, err)

	default:
		return fmt.Errorf("%w: failed creating %T: %s", trails.ErrUnexpected, value, err)
	}
}

// Delete archives or soft deletes the database record for value.
func (db *DB) Delete(value any) error {
	if db.db.Error != nil {
		return db.db.Error
	}

	res := db.db.Delete(value)
	if errors.Is(res.Error, schema.ErrUnsupportedDataType) {
		return fmt.Errorf("%w: cannot parse table name from %T", trails.ErrMissingData, value)
	}

	if res.Error != nil {
		return fmt.Errorf("%w: failed deleting %T: %s", trails.ErrUnexpected, value, res.Error)
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: %T", trails.ErrNotFound, value)
	}

	return nil
}

// Exec executes SQL query sql, passing values to it.
//
// If the query executed does not affect any records, Exec return ErrNotFound.
// There are many use cases where the caller out to specifically ignore this error,
// since the execution may not change existing records.
//
// Exec does not write any data resulting from the query into Go values.
func (db *DB) Exec(sql string, values ...any) error {
	if db.db.Error != nil {
		return db.db.Error
	}

	var err error
	values, err = unwrap(values...)
	if err != nil && !errors.Is(err, errNilArg) {
		return err
	}

	res := db.db.Exec(sql, values...)
	if res.Error != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: exec failed to affect any rows", trails.ErrNotFound)
	}

	return nil
}

// Exists asserts whether any record matches the current query.
func (db *DB) Exists() (bool, error) {
	if db.db.Error != nil {
		return false, db.db.Error
	}

	var exists bool
	// NOTE(dlk): This is weird and can't explain well why *gorm.DB.Session
	// is necessary in this instance.
	// Without it, GORM fails to render the current query as a sub-query.
	//
	// FIXME(dlk): Scan produces a SELECT * when a bare SELECT is sufficient,
	// some hack possible to get that marginal optimization?
	err := db.db.Raw("SELECT EXISTS(?)", db.db.Session(safeGORMSession)).Scan(&exists).Error
	if err != nil {
		err = fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
		return false, err
	}

	return exists, nil
}

// Find retrieves all records matching the current query
// and stores them in dest.
//
// If dest is not a valid type for the table queried,
// then ErrNotValid returns.
// If no matches are found, Find returns ErrNotFound.
func (db *DB) Find(dest any) (err error) {
	badDest := fmt.Errorf("%w: %T cannot be scanned into", trails.ErrNotValid, dest)
	defer func() {
		if r := recover(); r != nil {
			err = badDest
		}
	}()

	if db.db.Error != nil {
		return db.db.Error
	}

	res := db.db.Find(dest)
	err = res.Error
	if err != nil && errSQLScan.MatchString(err.Error()) {
		return badDest
	}

	if err != nil && errSQLSyntax.MatchString(err.Error()) {
		return fmt.Errorf("%w: %s", trails.ErrNotValid, err)
	}

	if err != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	return nil
}

// First retrieves a single record from the database matching the query
// and stores it in dest.
//
// If no matches are found, First returns ErrNotFound.
func (db *DB) First(dest any) error {
	if db.db.Error != nil {
		return db.db.Error
	}

	// FIXME(dlk): Specifically with sql.Null* types,
	// First is not overwriting previously set values
	// when the new sql.Null* is its zero-value.
	//
	// Cf. *postgres_test.DBTestSuite.TestFirst_NullTimeBug
	err := db.db.First(dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// FIXME(dlk): add %T from db.db.Stmt to give more context
		return fmt.Errorf("%w", trails.ErrNotFound)
	}

	if err != nil && errSQLSyntax.MatchString(err.Error()) {
		return fmt.Errorf("%w: %s", trails.ErrNotValid, err)
	}

	if err != nil {
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	return nil
}

// Paged turns the results of the current query into a paginated version: PagedData.
//
// FIXME(dlk): Paged is incompatible with Table for the time being
// and reurns ErrUnaddressable since the type queried data ought to be coerced into
// cannot be ascertained with reflection.
func (db *DB) Paged(page, perPage int64) (pd PagedData, err error) {
	defer func() {
		// NOTE(dlk): This method uses reflect and so can panic.
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: Paged panicked: %s", trails.ErrUnexpected, r)
			pd = PagedData{}
		}
	}()

	if db.db.Error != nil {
		return PagedData{}, db.db.Error
	}

	model := db.DB().Statement.Model
	if model == nil {
		err = fmt.Errorf("%w: must use Model with Paged", trails.ErrUnaddressable)
		return PagedData{}, err
	}

	reflectType := reflect.TypeOf(db.DB().Statement.Model).Elem()
	if reflectType.Kind() != reflect.Slice {
		model = reflect.New(reflect.SliceOf(reflectType)).Interface()
	}

	pd.Items = model
	pd.Page = max(1, page)
	pd.PerPage = max(1, perPage)

	var totalRecords int64
	err = db.db.Session(safeGORMSession).Count(&totalRecords).Error
	if err != nil {
		return PagedData{}, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	offset := int((pd.Page - 1) * pd.PerPage)
	err = db.db.Limit(int(pd.PerPage)).Offset(offset).Find(pd.Items).Error
	if err != nil && !errors.Is(err, trails.ErrNotFound) {
		return PagedData{}, fmt.Errorf("%w: %s", trails.ErrUnexpected, err)
	}

	// NOTE(dlk): use math/big for accurate float64 division.
	totalPages := new(big.Float).SetInt(big.NewInt(totalRecords))
	perPageFl := new(big.Float).SetInt(big.NewInt(perPage))

	// NOTE(dlk): guard divison by zero.
	zero := big.NewFloat(0)
	if totalPages.Cmp(zero) != 0 && perPageFl.Cmp(zero) != 0 {
		totalPages.Quo(totalPages, perPageFl)
	}

	// NOTE(dlk): We want rounding up, but Int64 rounds towards zero
	// and RoundingMode doesn't change this.
	// So, add one when it truncates incorrectly to get rounding up to the ceiling.
	var acc big.Accuracy
	pd.TotalPages, acc = totalPages.Int64()
	if acc == big.Below {
		pd.TotalPages += 1
	}

	pd.TotalItems = totalRecords

	return pd, nil
}

// Raw executes sql, passing values to it, and scans the results into dest.
func (db *DB) Raw(dest any, sql string, values ...any) error {
	if db.db.Error != nil {
		return db.db.Error
	}

	var err error
	values, err = unwrap(values...)
	if err != nil && !errors.Is(err, errNilArg) {
		return err
	}

	err = db.db.Raw(sql, values...).Scan(dest).Error
	if err != nil && errSQLSyntax.MatchString(err.Error()) {
		return fmt.Errorf("%w: %s", trails.ErrNotValid, err)
	}

	if err != nil && errSQLUnaddressable.MatchString(err.Error()) {
		return fmt.Errorf("%w: %s", trails.ErrUnaddressable, err)
	}

	if err != nil {
		return fmt.Errorf("%w: failed scanning results: %s", trails.ErrUnexpected, err)
	}

	return nil
}

// Updates replaces existing data on all records matching the query with values.
//
// If no records are updated, ErrNotFound returns.
// The caller ought to specifically handle this error
// when its expected a query may not mutate records.
//
// NB: It is tempting to re-use the same identifier in an Update & First flow.
// Generally, it is safer to zero out the identifier before re-using it this way.
// Cf. postgres_test.DBTestSuite.TestFirst_NullTimeBug.
func (db *DB) Update(values Updates) error {
	if db.db.Error != nil {
		return db.db.Error
	}

	if err := values.valid(); err != nil {
		return err
	}

	res := db.db.Updates(map[string]any(values))
	switch {
	case res.RowsAffected == 0 && res.Error == nil:
		return fmt.Errorf("%w", trails.ErrNotFound)

	case res.Error == nil:
		return nil

	case errUniqViolation.MatchString(res.Error.Error()):
		return fmt.Errorf("%w: %s", trails.ErrExists, res.Error)

	default:
		return fmt.Errorf("%w: %s", trails.ErrUnexpected, res.Error)
	}
}

// **************************************************************************
// QUERY BUILDING METHODS
//
// Query building methods initiate a query and then add clauses to it
// until a finisher method is called.
// The caller can chain methods.
// There is no required sort order,
// but conventions dictate acceptable patterns for which methods are called first.
//
// **************************************************************************

// Distinct adds a DISTINCT clause to the current query.
// Column can be an empty string, which is the equivalent of all columns, i.e.: *.
//
// FIXME(dlk): There's no current use case for multiple distincts in a query across our codebases.
// But, this ought to accept ...string since that's a perfectly reasonable future need.
// Take a look at this implementation:
//
//	func (db *DB) Distinct(columns ...string) *DB {
//		var v []any
//		for _, c := range columns {
//			v = append(v, c)
//		}
//		return &DB{db.db.Distinct(v...)}
//
// For whatever reason, GORM ends up applying none of the strings to the query.
// Switching to ...any doesn't resolve the issue.
func (db *DB) Distinct(column string) *DB {
	if column == "" { // NOTE(dlk): not necessary, but makes usage explicit.
		column = "*"
	}

	return &DB{db.db.Distinct(column)}
}

// Group applies a GROUP BY clause to the current query.
func (db *DB) Group(name string) *DB { return &DB{db: db.db.Group(name)} }

// Joins applies the JOIN statement query and args to the current query.
// args can include a *postgres.DB, that is, a subquery.
func (db *DB) Joins(query string, args ...any) *DB {
	var err error
	args, err = unwrap(args...)
	if err != nil && !errors.Is(err, errNilArg) {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(err)
		return &DB{db: gdb}
	}

	return &DB{db: db.db.Joins(query, args...)}
}

// Limit applies a LIMIT clause to the current query.
func (db *DB) Limit(limit int) *DB {
	// NOTE(dlk): GORM interprets negatives by not applying a LIMIT clause.
	// PostgreSQL errors on negative numbers:
	//     ERROR:  LIMIT must not be negative
	//
	// This Limit mirrors PostgreSQL, not GORM.
	if limit < 0 {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(fmt.Errorf("%w: limit must not be negative", trails.ErrNotValid))
		return &DB{db: gdb}
	}

	return &DB{db: db.db.Limit(limit)}
}

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
// Calling Model multiple times or in conjunction with Table is undefined behavior.
// Experience indicates the last call in the chain sets the table - buyer beware.
func (db *DB) Model(model any) *DB {
	// FIXME(dlk): gorm.Statement.Model is set in *gorm.DB.Model call
	// so could be used to have more definitive behavior around multiple *postgres.DB.Model calls.

	// TODO(dlk): *postgres.DB.Model calling *gorm.DB.Table instead of *gorm.DB.Model
	// has value for avoiding *gorm.DB.Model's behavior of applying the primary key of model,
	// when it is set, and this causing unexpected bugs.
	//
	// An initial experiment using *gorm.DB.Statement.Parse(model)
	// to extract the table name off *gorm.DB.Statement.Table
	// broke more than expected, such as Exists, Joins and Paged.
	//
	// More concetrated effort here could enable this simplification.
	return &DB{db: db.db.Model(model)}
}

// Offset applies an OFFSET clause to the current query.
func (db *DB) Offset(offset int) *DB {
	// NOTE(dlk): GORM interprets negatives by not applying an OFFSET clause.
	// PostgreSQL errors on negative numbers:
	//     ERROR:  OFFSET must not be negative
	//
	// This Offset  mirrors PostgreSQL, not GORM.
	if offset < 0 {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(fmt.Errorf("%w: offset must not be negative", trails.ErrNotValid))
		return &DB{db: gdb}
	}

	return &DB{db: db.db.Offset(offset)}
}

// Or applies an OR clause to the current query.
func (db *DB) Or(query any, args ...any) *DB {
	if len(args) > 1 {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(fmt.Errorf("%w: Or supports one or none args", trails.ErrNotValid))
		return &DB{db: gdb}
	}

	var err error
	args, err = unwrap(args...)
	if err != nil && !errors.Is(err, errNilArg) {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(err)
		return &DB{db: gdb}
	}

	q, err := unwrap(query)
	if err != nil {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(err)
		return &DB{db: gdb}
	}

	return &DB{db: db.db.Or(q[0], args...)}
}

// Order applies an ORDER BY clause to the current query.
func (db *DB) Order(order string) *DB {
	// FIXME(dlk): college-try has a use case that Order could support like:
	//
	//	Order("users.business_coordinates <@> ? ASC", query.LatLong())
	//
	// Currently, college-try does this:
	//
	//	q.DB().Clauses(clause.OrderBy{Expression: clause.Expr{
	//		SQL:  "users.business_coordinates <@> ? ASC",
	//			Vars: []any{query.LatLong()},
	//	}})
	return &DB{db: db.db.Order(order)}
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
	for _, scope := range scopes {
		// NOTE(dlk): naively passing db or db.db means
		// scope applies itself to the current query
		// instead of on the new query for the preload association.
		clean := NewDB(db.db.Session(&gorm.Session{NewDB: true}))
		resolved = append(resolved, scope(clean).DB())
	}

	return &DB{db: db.db.Preload(association, resolved...)}
}

// Scope applies the scope to the existing query.
// Review [Scope] for more details.
func (db *DB) Scope(scope Scope) *DB {
	return &DB{db: db.db.Scopes(func(dbx *gorm.DB) *gorm.DB {
		return scope(NewDB(dbx)).DB()
	})}
}

// Select applies a SELECT statement to the current query.
func (db *DB) Select(columns ...string) *DB { return &DB{db: db.db.Select(columns)} }

// Table defines which database table to query for the current query.
// Table is similar to Model but allows for explicit definition of the table.
// This can be helpful for dealing with table aliases, subqueries and similar temporary uses.
//
// Calling Table multiple times or in conjuction with Model
// in the same query chain is undefined behavior.
// Experience indicates the last call between Table and Model in the chain sets the table - buyer beware.
func (db *DB) Table(name string) *DB { return &DB{db: db.db.Table(name)} }

// Unscoped includes archived, soft deleted records in the current query.
func (db *DB) Unscoped() *DB { return &DB{db: db.db.Unscoped()} }

// Where applies the query fragment or subquery to the current query
// as a WHERE or AND clause.
//
// Where supports one or none args.
// If more than one arg is passed, finisher methods will return trailsErrNotValid.
func (db *DB) Where(query any, args ...any) *DB {
	if len(args) > 1 {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(fmt.Errorf("%w: Where supports one or none args", trails.ErrNotValid))
		return &DB{db: gdb}
	}

	var err error
	args, err = unwrap(args...)
	if err != nil && !errors.Is(err, errNilArg) {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(err)
		return &DB{db: gdb}
	}

	q, err := unwrap(query)
	if err != nil {
		gdb := db.DB().Session(safeGORMSession)
		_ = gdb.AddError(err)
		return &DB{db: gdb}
	}

	return &DB{db.db.Where(q[0], args...)}
}

// **************************************************************************
// TRANSACTION METHODS
//
// These methods control database transactions.
// **************************************************************************

// Begin initializes a database transaction.
func (db *DB) Begin(opts ...*sql.TxOptions) *DB {
	return &DB{db: db.db.Begin(opts...)}
}

// Commit completes the current transaction,
// applying any state changes and making them visible to other database connections.
func (db *DB) Commit() error {
	if db.db.Error != nil {
		return db.db.Error
	}

	if err := db.db.Commit().Error; err != nil {
		err = fmt.Errorf("%w: failed committing tx: %s", trails.ErrUnexpected, err)
		return err
	}

	return nil
}

// Rollback reverts the current transaction.
// If no transaction is open, Rollback returns an error.
func (db *DB) Rollback() error {
	err := db.db.Rollback().Error
	if err != nil {
		return fmt.Errorf("%w: failed rolling back tx: %s", trails.ErrUnexpected, err)
	}

	return nil
}

// **************************************************************************
// HELPERS
//
// **************************************************************************
// unwrap converts any custom postgres types that are troublesome for GORM into types it can handle.
// unwrap ought to be applied to parameters of any type.
// unwrap returns an error in exceptional circumstances.
//
// If unwrapping a parameter uncovers some error, unwrap returns the error.
// Notably, if a *DB is passed as a parameter,
// and that *DB is in an error state, that fact is surfaced.
// This enables a *DB method to return early and prevent partial queries from running.
func unwrap(args ...any) ([]any, error) {
	var err error
	res := make([]any, len(args))
	for i, arg := range args {
		// NOTE(dlk): other custom types that obfuscate GORM types
		// go here in order to expose the appropriate GORM type
		switch v := arg.(type) {
		case *DB:
			gdb := v.DB()
			if err := gdb.Error; err != nil {
				err = gdb.Error
			}
			res[i] = gdb

		case nil:
			res[i] = arg
			err = errors.Join(err, trails.ErrNotValid, errNilArg)

		default:
			res[i] = arg
		}
	}

	return res, err
}
