package db

// Pagination holds common pagination parameters.
type Pagination struct {
	Page      int    `json:"page"` // starts from 1
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"` // "asc" or "desc"
}

// ListResult is a generic paginated result.
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int64 `json:"total_count"`
}

// ListQuery holds generic filter + pagination parameters for list operations.
type ListQuery struct {
	Filters map[string]any
	Pagination
}

// PaginationToOffsetLimit converts Pagination to offset and limit with defaults.
func PaginationToOffsetLimit(p Pagination) (offset int32, limit int32) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return int32((page - 1) * pageSize), int32(pageSize)
}
