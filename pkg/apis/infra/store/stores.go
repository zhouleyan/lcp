package store

import (
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db"
)

// NewStores creates all infra store implementations using the shared database connection.
func NewStores(database *db.DB) infra.Stores {
	return infra.Stores{
		Host:        NewPGHostStore(database.Pool, database.Queries),
		Environment: NewPGEnvironmentStore(database.Pool, database.Queries),
		Region:         NewPGRegionStore(database.Queries),
		Site:           NewPGSiteStore(database.Queries),
		Location:       NewPGLocationStore(database.Queries),
		Rack:           NewPGRackStore(database.Queries),
	}
}
