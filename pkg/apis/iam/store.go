package iam

import (
	"context"

	"lcp.io/lcp/pkg/db"
)

// UserStore defines database operations on users.
type UserStore interface {
	Create(ctx context.Context, user *DBUser) (*DBUser, error)
	GetByID(ctx context.Context, id int64) (*DBUser, error)
	GetByUsername(ctx context.Context, username string) (*DBUser, error)
	GetByEmail(ctx context.Context, email string) (*DBUser, error)
	Update(ctx context.Context, user *DBUser) (*DBUser, error)
	Patch(ctx context.Context, id int64, user *DBUser) (*DBUser, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBUserWithNamespaces], error)
}

// NamespaceStore defines database operations on namespaces.
type NamespaceStore interface {
	Create(ctx context.Context, ns *DBNamespace) (*DBNamespace, error)
	GetByID(ctx context.Context, id int64) (*DBNamespace, error)
	GetByName(ctx context.Context, name string) (*DBNamespace, error)
	Update(ctx context.Context, ns *DBNamespace) (*DBNamespace, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error)
}

// UserNamespaceStore defines operations on user-namespace relationships.
type UserNamespaceStore interface {
	Add(ctx context.Context, rel *DBUserNamespace) (*DBUserNamespace, error)
	Remove(ctx context.Context, userID, namespaceID int64) error
	UpdateRole(ctx context.Context, rel *DBUserNamespace) (*DBUserNamespace, error)
	Get(ctx context.Context, userID, namespaceID int64) (*DBUserNamespace, error)
	ListByUserID(ctx context.Context, userID int64) ([]DBNamespaceWithRole, error)
	ListByNamespaceID(ctx context.Context, namespaceID int64) ([]DBUserWithRole, error)
}
