package db

import (
	"fmt"
	"strings"
)

// FieldOp defines how a filter field is matched in SQL.
type FieldOp int

const (
	Eq   FieldOp = iota // col = $N
	Like                 // col ILIKE '%' || $N || '%'
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
