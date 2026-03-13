package infra

import (
	"context"

	"lcp.io/lcp/pkg/db"
)

// HostStore defines database operations on hosts.
type HostStore interface {
	Create(ctx context.Context, host *DBHost) (*DBHost, error)
	GetByID(ctx context.Context, id int64) (*DBHostWithEnv, error)
	Update(ctx context.Context, host *DBHost) (*DBHost, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBHost, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	ListPlatform(ctx context.Context, query db.ListQuery) (*db.ListResult[DBHostPlatformRow], error)
	ListByWorkspaceID(ctx context.Context, wsID int64, query db.ListQuery) (*db.ListResult[DBHostWorkspaceRow], error)
	ListByNamespaceID(ctx context.Context, nsID int64, query db.ListQuery) (*db.ListResult[DBHostNamespaceRow], error)
	BindEnvironment(ctx context.Context, hostID, envID int64) error
	UnbindEnvironment(ctx context.Context, hostID int64) error
	GetWorkspaceIDByNamespaceID(ctx context.Context, nsID int64) (int64, error)
}

// EnvironmentStore defines database operations on environments.
type EnvironmentStore interface {
	Create(ctx context.Context, env *DBEnvironment) (*DBEnvironment, error)
	GetByID(ctx context.Context, id int64) (*DBEnvWithCounts, error)
	Update(ctx context.Context, env *DBEnvironment) (*DBEnvironment, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBEnvironment, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	ListPlatform(ctx context.Context, query db.ListQuery) (*db.ListResult[DBEnvPlatformRow], error)
	ListByWorkspaceID(ctx context.Context, wsID int64, query db.ListQuery) (*db.ListResult[DBEnvWorkspaceRow], error)
	ListByWorkspaceIDInherit(ctx context.Context, wsID int64, query db.ListQuery) (*db.ListResult[DBEnvWorkspaceInheritRow], error)
	ListByNamespaceID(ctx context.Context, nsID int64, query db.ListQuery) (*db.ListResult[DBEnvNamespaceRow], error)
	ListByNamespaceIDInherit(ctx context.Context, nsID int64, query db.ListQuery) (*db.ListResult[DBEnvNamespaceInheritRow], error)
	ListHostsByEnvID(ctx context.Context, envID int64, query db.ListQuery) (*db.ListResult[DBHostByEnvRow], error)
}

// RegionStore defines database operations on regions.
type RegionStore interface {
	Create(ctx context.Context, region *DBRegion) (*DBRegion, error)
	GetByID(ctx context.Context, id int64) (*DBRegionWithCounts, error)
	Update(ctx context.Context, region *DBRegion) (*DBRegion, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBRegion, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	CountChildSites(ctx context.Context, regionID int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBRegionListRow], error)
}

// SiteStore defines database operations on sites.
type SiteStore interface {
	Create(ctx context.Context, site *DBSite) (*DBSite, error)
	GetByID(ctx context.Context, id int64) (*DBSiteWithDetails, error)
	Update(ctx context.Context, site *DBSite) (*DBSite, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBSite, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	CountChildLocations(ctx context.Context, siteID int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBSiteListRow], error)
}

// LocationStore defines database operations on locations.
type LocationStore interface {
	Create(ctx context.Context, location *DBLocation) (*DBLocation, error)
	GetByID(ctx context.Context, id int64) (*DBLocationWithDetails, error)
	Update(ctx context.Context, location *DBLocation) (*DBLocation, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBLocation, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBLocationListRow], error)
	CountChildRacks(ctx context.Context, locationID int64) (int64, error)
}

// RackStore defines database operations on racks.
type RackStore interface {
	Create(ctx context.Context, rack *DBRack) (*DBRack, error)
	GetByID(ctx context.Context, id int64) (*DBRackWithDetails, error)
	Update(ctx context.Context, rack *DBRack) (*DBRack, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBRack, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBRackListRow], error)
}
