package postgres

// PagedData is returned from the Paged method.
// It contains paged database records and pagination metadata.
type PagedData struct {
	Items      any   `json:"items"`
	Page       int64 `json:"page"`
	PerPage    int64 `json:"perPage"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int64 `json:"totalPages"`
}
