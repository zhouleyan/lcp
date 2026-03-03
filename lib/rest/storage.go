package rest

import (
	"context"

	"lcp.io/lcp/lib/runtime"
)

// ValidateObjectFunc is kept for backward compatibility.
// Deprecated: validation should be internal to Storage implementations.
type ValidateObjectFunc func(ctx context.Context, obj runtime.Object) error

// Getter handles GET for a single resource.
type Getter interface {
	Get(ctx context.Context, id string) (runtime.Object, error)
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
	Update(ctx context.Context, id string, obj runtime.Object, options *UpdateOptions) (runtime.Object, error)
}

// Patcher handles PATCH for partial updates.
type Patcher interface {
	Patch(ctx context.Context, id string, obj runtime.Object, options *PatchOptions) (runtime.Object, error)
}

// Deleter handles DELETE for a single resource.
type Deleter interface {
	Delete(ctx context.Context, id string, options *DeleteOptions) error
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
