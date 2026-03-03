package store

import (
	"context"

	libstore "lcp.io/lcp/lib/store"
)

// NamespaceStore defines operations on namespaces.
type NamespaceStore interface {
	Create(ctx context.Context, ns *Namespace) (*Namespace, error)
	GetByID(ctx context.Context, id int64) (*Namespace, error)
	GetByName(ctx context.Context, name string) (*Namespace, error)
	Update(ctx context.Context, ns *Namespace) (*Namespace, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query libstore.ListQuery) (*libstore.ListResult[NamespaceWithOwner], error)
}

// UserNamespaceStore defines operations on user-namespace relationships.
type UserNamespaceStore interface {
	Add(ctx context.Context, rel *UserNamespaceRole) (*UserNamespaceRole, error)
	Remove(ctx context.Context, userID, namespaceID int64) error
	UpdateRole(ctx context.Context, rel *UserNamespaceRole) (*UserNamespaceRole, error)
	Get(ctx context.Context, userID, namespaceID int64) (*UserNamespaceRole, error)
	ListByUserID(ctx context.Context, userID int64) ([]NamespaceWithRole, error)
	ListByNamespaceID(ctx context.Context, namespaceID int64) ([]UserWithRole, error)
}
