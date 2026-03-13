package store

import (
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db"
)

// NewStores creates all infra store implementations using the shared database connection.
func NewStores(database *db.DB) infra.Stores {
	ipBinder := NewPGIPBinder(database.Pool, database.Queries)
	return infra.Stores{
		Host:        NewPGHostStore(database.Pool, database.Queries, ipBinder),
		Environment: NewPGEnvironmentStore(database.Pool, database.Queries),
		Region:         NewPGRegionStore(database.Queries),
		Site:           NewPGSiteStore(database.Queries),
		Location:       NewPGLocationStore(database.Queries),
		Rack:           NewPGRackStore(database.Queries),
		IPBinder:       ipBinder,
	}
}
