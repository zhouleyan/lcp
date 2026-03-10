package apis

import (
	"context"

	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	dashboardv1 "lcp.io/lcp/pkg/apis/dashboard/v1"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	"lcp.io/lcp/pkg/db"
)

// Result holds the outputs of API module initialization.
type Result struct {
	Groups []*rest.APIGroupInfo
}

// NewAPIGroupInfos assembles all API modules, syncs permissions for all groups,
// and returns the aggregated result.
func NewAPIGroupInfos(ctx context.Context, database *db.DB) Result {
	// --- IAM module ---
	iamResult := iamv1.NewIAMModule(ctx, database)

	// --- Dashboard module ---
	dashboardResult := dashboardv1.NewDashboardModule(database)

	groups := []*rest.APIGroupInfo{iamResult.Group, dashboardResult.Group}

	// Sync permissions for ALL modules centrally
	iamv1.SyncAllPermissions(ctx, database, groups)

	return Result{
		Groups: groups,
	}
}

// NewAuthorizer creates a fully-wired Authorizer from API group definitions and database.
func NewAuthorizer(database *db.DB, groups []*rest.APIGroupInfo) *filters.Authorizer {
	return iamv1.NewAuthorizer(database, groups)
}

// NewOIDCProvider creates the OIDC provider with all internal store wiring.
// Returns nil if OIDC is not configured (no key files).
func NewOIDCProvider(database *db.DB, cfg *config.OIDCConfig) *oidc.Provider {
	return iamv1.NewOIDCProvider(database, cfg)
}
