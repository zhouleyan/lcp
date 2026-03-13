package o11y

import (
	"context"

	"lcp.io/lcp/pkg/db"
)

// EndpointStore defines database operations on o11y endpoints.
type EndpointStore interface {
	Create(ctx context.Context, ep *DBEndpoint) (*DBEndpoint, error)
	GetByID(ctx context.Context, id int64) (*DBEndpoint, error)
	Update(ctx context.Context, ep *DBEndpoint) (*DBEndpoint, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBEndpoint, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBEndpoint], error)
}

// Stores holds all o11y store instances.
type Stores struct {
	Endpoint EndpointStore
}
