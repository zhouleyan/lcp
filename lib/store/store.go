package store

import "context"

// Store is the top-level interface aggregating all sub-stores.
type Store interface {
	Users() UserStore
	Namespaces() NamespaceStore
	UserNamespaces() UserNamespaceStore
	WithTx(ctx context.Context, fn func(Store) error) error
	Close()
}
