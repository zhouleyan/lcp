package store

import (
	"lcp.io/lcp/pkg/apis/o11y"
	"lcp.io/lcp/pkg/db"
)

// NewStores creates all o11y store implementations using the shared database connection.
func NewStores(database *db.DB) o11y.Stores {
	return o11y.Stores{
		Endpoint: NewPGEndpointStore(database.Pool, database.Queries),
	}
}
