package rest

import (
	"context"

	"lcp.io/lcp/lib/runtime"
)

// Storage is a marker interface for resource storage implementations.
// Concrete types should implement one or more of:
// Getter, Lister, Creator, Updater, Patcher, Deleter, CollectionDeleter.
type Storage interface{}

// ObjectCreator returns a new empty instance of the resource type managed
// by this storage. It is used by handlers to provide a decode target when
// deserializing request bodies.
type ObjectCreator interface {
	NewObject() runtime.Object
}

// Getter handles GET for a single resource.
type Getter interface {
	Get(ctx context.Context, options *GetOptions) (runtime.Object, error)
}

// Lister handles GET for a collection (with filtering/pagination/sorting).
type Lister interface {
	List(ctx context.Context, options *ListOptions) (runtime.Object, error)
}

// Creator handles POST to create a resource.
type Creator interface {
	Create(ctx context.Context, obj runtime.Object, options *CreateOptions) (runtime.Object, error)
}

// Updater handles PUT to fully replace a resource.
type Updater interface {
	Update(ctx context.Context, obj runtime.Object, options *UpdateOptions) (runtime.Object, error)
}

// Patcher handles PATCH for partial updates.
type Patcher interface {
	Patch(ctx context.Context, obj runtime.Object, options *PatchOptions) (runtime.Object, error)
}

// Deleter handles DELETE for a single resource.
type Deleter interface {
	Delete(ctx context.Context, options *DeleteOptions) error
}

// CollectionDeleter handles batch DELETE (by explicit ID list).
type CollectionDeleter interface {
	DeleteCollection(ctx context.Context, ids []string, options *DeleteOptions) (*DeletionResult, error)
}

// StandardStorage combines all operations.
type StandardStorage interface {
	Getter
	Lister
	Creator
	Updater
	Patcher
	Deleter
	CollectionDeleter
}

// DeletionResult holds batch delete results.
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
}

// DeleteCollectionRequest is the request body for batch delete operations.
type DeleteCollectionRequest struct {
	IDs []string `json:"ids"`
}
