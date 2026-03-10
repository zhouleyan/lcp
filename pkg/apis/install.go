package apis

import (
	"context"

	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	dashboardv1 "lcp.io/lcp/pkg/apis/dashboard/v1"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	"lcp.io/lcp/pkg/db"
)

// Result holds the outputs of API module initialization.
type Result struct {
	Groups []*rest.APIGroupInfo
}

// NewAPIGroupInfos assembles all API modules and returns the aggregated result.
func NewAPIGroupInfos(ctx context.Context, database *db.DB) Result {
	// --- IAM module ---
	iamResult := iamv1.NewIAMModule(ctx, database)

	// --- Dashboard module ---
	dashboardResult := dashboardv1.NewDashboardModule(database)

	return Result{
		Groups: []*rest.APIGroupInfo{iamResult.Group, dashboardResult.Group},
	}
}

// NewAuthorizer creates a fully-wired Authorizer from API group definitions and database.
func NewAuthorizer(database *db.DB, groups []*rest.APIGroupInfo) *filters.Authorizer {
	rbStore := iamstore.NewPGRoleBindingStore(database.Pool, database.Queries)
	nsStore := iamstore.NewPGNamespaceStore(database.Pool, database.Queries)
	return iam.NewAuthorizer(rbStore, nsStore, groups)
}

// NewOIDCProvider creates the OIDC provider with all internal store wiring.
// Returns nil if OIDC is not configured (no key files).
func NewOIDCProvider(database *db.DB, cfg *config.OIDCConfig) *oidc.Provider {
	if cfg.PrivateKeyFile == "" || cfg.PublicKeyFile == "" {
		logger.Infof("OIDC not configured (no key files), authentication disabled")
		return nil
	}

	providerCfg, err := oidc.ParseConfig(cfg)
	if err != nil {
		logger.Fatalf("invalid OIDC config: %v", err)
	}

	keySet, err := oidc.LoadKeySet(cfg.PrivateKeyFile, cfg.PublicKeyFile)
	if err != nil {
		logger.Fatalf("cannot load OIDC keys: %v", err)
	}

	userStore := iamstore.NewPGUserStore(database.Queries)
	refreshStore := iamstore.NewPGRefreshTokenStore(database.Queries)

	provider := oidc.NewProvider(providerCfg, keySet,
		iam.NewUserLookupAdapter(userStore),
		iam.NewRefreshTokenAdapter(refreshStore),
	)
	provider.SetClients(oidc.ParseClients(cfg.Clients))

	logger.Infof("OIDC provider initialized (issuer=%s)", cfg.Issuer)
	return provider
}
