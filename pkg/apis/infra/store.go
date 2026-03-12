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
}

// HostAssignmentStore defines database operations on host assignments.
type HostAssignmentStore interface {
	Assign(ctx context.Context, hostID int64, wsID, nsID *int64) (*DBHostAssignment, error)
	UnassignWorkspace(ctx context.Context, hostID int64, wsID int64) error
	UnassignNamespace(ctx context.Context, hostID int64, nsID int64) error
	ListByHostID(ctx context.Context, hostID int64) ([]DBAssignmentRow, error)
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
	ListByNamespaceID(ctx context.Context, nsID int64, query db.ListQuery) (*db.ListResult[DBEnvNamespaceRow], error)
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
	ListByRegionID(ctx context.Context, regionID int64, query db.ListQuery) (*db.ListResult[DBSiteByRegionRow], error)
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
	ListBySiteID(ctx context.Context, siteID int64, query db.ListQuery) (*db.ListResult[DBLocationBySiteRow], error)
}
