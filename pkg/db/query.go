package db

import (
	"fmt"
	"strings"
)

// FieldOp defines how a filter field is matched in SQL.
type FieldOp int

const (
	Eq   FieldOp = iota // col = $N
	Like                // col ILIKE '%' || $N || '%'
)

// Field maps a filter key to a SQL column and match operation.
type Field struct {
	Column string
	Op     FieldOp
}

// ListSpec declares the filterable/sortable fields for a list query.
// Column values are hardcoded (not from user input), ensuring SQL injection safety.
type ListSpec struct {
	Fields      map[string]Field
	DefaultSort string // e.g. "u.created_at"
}

// BuildWhereClause builds a parameterized WHERE clause from filters and a ListSpec.
// argStart is the 1-based index of the first placeholder ($1, $2, ...).
// Returns the clause string (including " WHERE ") and the corresponding args slice.
// If no filters match, returns an empty string and nil args.
func BuildWhereClause(filters map[string]any, spec ListSpec, argStart int) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any
	idx := argStart

	for key, val := range filters {
		field, ok := spec.Fields[key]
		if !ok {
			continue
		}
		switch field.Op {
		case Eq:
			conditions = append(conditions, fmt.Sprintf("%s = $%d", field.Column, idx))
			args = append(args, val)
			idx++
		case Like:
			s, ok := val.(string)
			if !ok {
				continue
			}
			conditions = append(conditions, fmt.Sprintf("%s ILIKE '%%' || $%d || '%%'", field.Column, idx))
			args = append(args, EscapeLike(s))
			idx++
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// BuildOrderBy builds an ORDER BY clause from sortBy/sortOrder and the ListSpec.
// Only columns declared in spec.Fields are allowed; unknown values fall back to DefaultSort.
func BuildOrderBy(sortBy, sortOrder string, spec ListSpec) string {
	order := "DESC"
	if strings.EqualFold(sortOrder, "asc") {
		order = "ASC"
	}

	// Validate sortBy against known fields
	if sortBy != "" {
		if f, ok := spec.Fields[sortBy]; ok {
			return fmt.Sprintf(" ORDER BY %s %s", f.Column, order)
		}
	}

	// Fallback to default
	return fmt.Sprintf(" ORDER BY %s %s", spec.DefaultSort, order)
}

// Pagination holds common pagination and sorting parameters.
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
