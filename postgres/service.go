package postgres

import (
	"math"

	"gorm.io/gorm"
)

// DatabaseService sets up the interface to be used at the handler/middelware level. These should be straightforward
// calls that allow us to skip creating a procedure method for the most basic database interactions. At the procedural
// layer, the *gorm.DB struct is available directly for more complex composition. This has the intended functionality
// that we are not testing the database in handlers, while it is tested directly at the procedural layer.
type DatabaseService interface {
	CountByQuery(model any, query map[string]any) (int64, error)
	FetchByQuery(models any, query string, params []any) error
	FindByID(model any, ID any) error
	FindByQuery(model any, query map[string]any) error
	PagedByQuery(models any, query string, params []any, order string, page int, perPage int, preloads ...string) (PagedData, error)
	PagedByQueryFromSession(models any, session *gorm.DB, page int, perPage int) (PagedData, error)
}

// PagedData is returned from the PagedByQuery method. It contains paged database records and pagination metadata.
type PagedData struct {
	Items      any   `json:"items"`
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
