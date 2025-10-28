package postgres

// DatabaseService sets up the interface to be used at the handler/middelware level. These should be straightforward
// calls that allow us to skip creating a procedure method for the most basic database interactions. At the procedural
// layer, the *gorm.DB struct is available directly for more complex composition. This has the intended functionality
// that we are not testing the database in handlers, while it is tested directly at the procedural layer.
//
// Deprecated: use *DB
type DatabaseService interface {
	CountByQuery(model any, query map[string]any) (int64, error)
	FetchByQuery(models any, query string, params []any) error
	FindByID(model any, ID any) error
	FindByQuery(model any, query map[string]any) error
	PagedByQuery(models any, query string, params []any, order string, page int64, perPage int64, preloads ...string) (PagedData, error)
}

// CountByQuery recives a database model and query and fetches a count for the given params.
//
// Deprecated: chain queries with db.Model(model).Where(...).Count()
func (db *DB) CountByQuery(model any, query map[string]any) (int64, error) {
	q := db.Model(model)
	for k, v := range query {
		q = q.Where(k, v)
	}

	return q.Count()
}

// FetchByQuery receives a slice of database models as a pointer and fetches all records matching the query.
//
// Deprecated: chain queries with db.Where(...).Find(models)
func (db *DB) FetchByQuery(models any, query string, params []any) error {
	return db.DB().Where(query, params...).Find(models).Error
}

// FindByID receives a database model as a pointer and fetches it using the primary ID.
//
// Deprecated: try db.Where("id = ?", ID).First(model)
func (db *DB) FindByID(model any, ID any) error { return db.Where("id = ?", ID).First(model) }

// FindByQuery receives a database model as a pointer and fetches it using the given query.
//
// Deprecated: chain queries with db.Where(...).First(model)
func (db *DB) FindByQuery(model any, query map[string]any) error {
	q := db
	for k, v := range query {
		q = q.Where(k, v)
	}

	return q.First(model)
}

// Insert receives a database model and inserts it into the database.
//
// Deprecated: use db.Create(model)
func (db *DB) Insert(model any) error { return db.Create(model) }

// PagedByQuery receives a slice of database models and paging information to build a paged database query.
//
// Deprecated: use db.Model(models).Preload(...).Where(...).Order(...).Paged(...)
func (db *DB) PagedByQuery(models any, query string, params []any, order string, page int64, perPage int64, preloads ...string) (PagedData, error) {
	q := db.Model(models).Where(query, params...).Order(order)
	for _, preload := range preloads {
		q = q.Preload(preload)
	}

	return q.Paged(page, perPage)
}
