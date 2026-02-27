package store

import "context"

// UserNamespaceStore defines operations on user-namespace relationships.
type UserNamespaceStore interface {
	Add(ctx context.Context, params AddUserNamespaceParams) (*UserNamespaceRole, error)
	Remove(ctx context.Context, userID, namespaceID int64) error
	UpdateRole(ctx context.Context, params UpdateRoleParams) (*UserNamespaceRole, error)
	Get(ctx context.Context, userID, namespaceID int64) (*UserNamespaceRole, error)
	ListByUserID(ctx context.Context, userID int64) ([]NamespaceWithRole, error)
	ListByNamespaceID(ctx context.Context, namespaceID int64) ([]UserWithRole, error)
}
