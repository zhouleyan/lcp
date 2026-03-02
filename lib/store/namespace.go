package store

import "context"

// NamespaceStore defines operations on namespaces.
type NamespaceStore interface {
	Create(ctx context.Context, ns *Namespace) (*Namespace, error)
	GetByID(ctx context.Context, id int64) (*Namespace, error)
	GetByName(ctx context.Context, name string) (*Namespace, error)
	Update(ctx context.Context, ns *Namespace) (*Namespace, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query ListQuery) (*ListResult[NamespaceWithOwner], error)
}
