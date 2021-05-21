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
	CountByQuery(model interface{}, query map[string]interface{}) (int64, error)
	FetchByQuery(models interface{}, query string, params []interface{}) error
	FindByID(model interface{}, ID interface{}) error
	FindByQuery(model interface{}, query map[string]interface{}) error
	PagedByQuery(models interface{}, query string, params []interface{}, order string, page int, perPage int, preloads ...string) (PagedData, error)
	PagedByQueryFromSession(models interface{}, session *gorm.DB, page int, perPage int) (PagedData, error)
}

// PagedData is returned from the PagedByQuery method. It contains paged database records and pagination metadata.
type PagedData struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PerPage    int         `json:"perPage"`
	TotalItems int64       `json:"totalItems"`
	TotalPages int         `json:"totalPages"`
}

// DatabaseServiceImpl satisfies the above DatabaseService interface.
type DatabaseServiceImpl struct {
	db *gorm.DB
}

// NewService hydrates the gorm database for the implementation struct methods.
func NewService(DB *gorm.DB) *DatabaseServiceImpl {
	return &DatabaseServiceImpl{db: DB}
}

// CountByQuery recives a database model and query and fetches a count for the given params.
func (service *DatabaseServiceImpl) CountByQuery(model interface{}, query map[string]interface{}) (int64, error) {
	count := int64(0)
	return count, service.db.Model(model).Where(query).Count(&count).Error
}

// FetchByQuery receives a slice of database models as a pointer and fetches all records matching the query.
func (service *DatabaseServiceImpl) FetchByQuery(models interface{}, query string, params []interface{}) error {
	return service.db.Where(query, params...).Find(models).Error
}

// FindByID receives a database model as a pointer and fetches it using the primary ID.
func (service *DatabaseServiceImpl) FindByID(model interface{}, ID interface{}) error {
	return service.db.First(model, ID).Error
}

// FindByQuery receives a database model as a pointer and fetches it using the given query.
func (service *DatabaseServiceImpl) FindByQuery(model interface{}, query map[string]interface{}) error {
	return service.db.Where(query).First(model).Error
}

// Insert receives a database model and inserts it into the database.
func (service *DatabaseServiceImpl) Insert(model interface{}) error {
	return service.db.Create(model).Error
}

// PagedByQuery receives a slice of database models and paging information to build a paged database query.
func (service *DatabaseServiceImpl) PagedByQuery(models interface{}, query string, params []interface{}, order string, page int, perPage int, preloads ...string) (PagedData, error) {
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
	if err := service.db.Where(query, params...).Model(models).Count(&totalRecords).Error; err != nil {
		return pd, err
	}

	// Calculate offset and conduct limited query
	offset := (page - 1) * perPage
	session := service.db
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
func (service *DatabaseServiceImpl) PagedByQueryFromSession(models interface{}, session *gorm.DB, page int, perPage int) (PagedData, error) {
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
