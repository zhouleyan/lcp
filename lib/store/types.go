package store

import (
	"time"

	"lcp.io/lcp/lib/db/generated"
)

// Pagination holds common pagination and sorting parameters.
type Pagination struct {
	Page      int    `json:"page"`       // starts from 1
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

// Base model types: aliases to sqlc-generated structs.
type User = generated.User
type Namespace = generated.Namespace
type UserNamespaceRole = generated.UserNamespace

// UserWithNamespaces extends User with associated namespace names.
type UserWithNamespaces struct {
	User
	NamespaceNames []string `json:"namespace_names"`
}

// NamespaceWithOwner extends Namespace with owner username.
type NamespaceWithOwner struct {
	Namespace
	OwnerUsername string `json:"owner_username"`
}

// NamespaceWithRole is a namespace with the user's role in it.
type NamespaceWithRole struct {
	Namespace
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// UserWithRole is a user with their role in a namespace.
type UserWithRole struct {
	User
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
