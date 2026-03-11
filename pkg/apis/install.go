package apis

import (
	"context"
	"net/http"

	libaudit "lcp.io/lcp/lib/audit"
	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	"lcp.io/lcp/pkg/apis/audit/store"
	auditv1 "lcp.io/lcp/pkg/apis/audit/v1"
	dashboardv1 "lcp.io/lcp/pkg/apis/dashboard/v1"
	"lcp.io/lcp/pkg/apis/iam"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	infrav1 "lcp.io/lcp/pkg/apis/infra/v1"
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

	// --- Audit module ---
	auditResult := auditv1.NewAuditModule(database)

	// --- Infra module ---
	infraResult := infrav1.NewInfraModule(database)

	groups := []*rest.APIGroupInfo{iamResult.Group, dashboardResult.Group, auditResult.Group, infraResult.Group}

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

// NewAuditWriter creates a fully-wired audit Writer from the database.
func NewAuditWriter(database *db.DB) *libaudit.Writer {
	sink := store.NewPGAuditLogStore(database.Queries)
	return libaudit.NewWriter(sink, libaudit.WriterConfig{})
}

// NewOIDCMux creates the OIDC public endpoint HTTP handler.
// Returns nil if provider is nil.
func NewOIDCMux(provider *oidc.Provider, auditLogger libaudit.Logger) http.Handler {
	if provider == nil {
		return nil
	}
	return iam.NewOIDCMux(provider, auditLogger)
}
