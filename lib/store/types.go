package store

import "time"

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

// User represents a user business model.
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Phone       string     `json:"phone"`
	AvatarUrl   string     `json:"avatar_url"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserWithNamespaces extends User with associated namespace names.
type UserWithNamespaces struct {
	User
	NamespaceNames []string `json:"namespace_names"`
}

// Namespace represents a namespace business model.
type Namespace struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	OwnerID     int64     `json:"owner_id"`
	Visibility  string    `json:"visibility"`
	MaxMembers  int32     `json:"max_members"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NamespaceWithOwner extends Namespace with owner username.
type NamespaceWithOwner struct {
	Namespace
	OwnerUsername string `json:"owner_username"`
}

// UserNamespaceRole represents a user's membership in a namespace.
type UserNamespaceRole struct {
	UserID      int64     `json:"user_id"`
	NamespaceID int64     `json:"namespace_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
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

// CreateUserParams holds parameters for creating a user.
type CreateUserParams struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	AvatarUrl   string `json:"avatar_url"`
	Status      string `json:"status"`
}

// UpdateUserParams holds parameters for updating a user.
type UpdateUserParams struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	AvatarUrl   string `json:"avatar_url"`
	Status      string `json:"status"`
}

// ListUsersParams holds parameters for listing users.
type ListUsersParams struct {
	Status      *string `json:"status,omitempty"`
	Username    *string `json:"username,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Pagination
}

// CreateNamespaceParams holds parameters for creating a namespace.
type CreateNamespaceParams struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnerID     int64  `json:"owner_id"`
	Visibility  string `json:"visibility"`
	MaxMembers  int32  `json:"max_members"`
	Status      string `json:"status"`
}

// UpdateNamespaceParams holds parameters for updating a namespace.
type UpdateNamespaceParams struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnerID     int64  `json:"owner_id"`
	Visibility  string `json:"visibility"`
	MaxMembers  int32  `json:"max_members"`
	Status      string `json:"status"`
}

// ListNamespacesParams holds parameters for listing namespaces.
type ListNamespacesParams struct {
	Status     *string `json:"status,omitempty"`
	Name       *string `json:"name,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
	OwnerID    *int64  `json:"owner_id,omitempty"`
	Pagination
}

// AddUserNamespaceParams holds parameters for adding a user to a namespace.
type AddUserNamespaceParams struct {
	UserID      int64  `json:"user_id"`
	NamespaceID int64  `json:"namespace_id"`
	Role        string `json:"role"`
}

// UpdateRoleParams holds parameters for updating a user's role in a namespace.
type UpdateRoleParams struct {
	UserID      int64  `json:"user_id"`
	NamespaceID int64  `json:"namespace_id"`
	Role        string `json:"role"`
}
