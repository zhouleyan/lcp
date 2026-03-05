package apis

import (
	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	"lcp.io/lcp/pkg/db"
)

// NewAPIGroupInfos creates all APIGroupInfo instances and the OIDC provider.
// Returns nil provider if OIDC is not configured (no key files).
func NewAPIGroupInfos(database *db.DB, cfg *config.Config) ([]*rest.APIGroupInfo, *oidc.Provider) {
	refreshTokenStore := iamstore.NewPGRefreshTokenStore(database.Queries)

	iamProvider := iam.NewRESTStorageProvider(
		iamstore.NewPGUserStore(database.Queries),
		iamstore.NewPGWorkspaceStore(database.Pool, database.Queries),
		iamstore.NewPGNamespaceStore(database.Pool, database.Queries),
		iamstore.NewPGUserWorkspaceStore(database.Queries),
		iamstore.NewPGUserNamespaceStore(database.Pool, database.Queries),
		refreshTokenStore,
	)

	var oidcProvider *oidc.Provider

	if cfg.OIDC.PrivateKeyFile != "" && cfg.OIDC.PublicKeyFile != "" {
		providerCfg, err := oidc.ParseConfig(&cfg.OIDC)
		if err != nil {
			logger.Fatalf("invalid OIDC config: %v", err)
		}

		keySet, err := oidc.LoadKeySet(cfg.OIDC.PrivateKeyFile, cfg.OIDC.PublicKeyFile)
		if err != nil {
			logger.Fatalf("cannot load OIDC keys: %v", err)
		}

		userLookup := iam.NewUserLookupAdapter(iamProvider.UserStore())
		refreshAdapter := iam.NewRefreshTokenAdapter(refreshTokenStore)

		oidcProvider = oidc.NewProvider(providerCfg, keySet, userLookup, refreshAdapter)
		oidcProvider.SetClients(oidc.ParseClients(cfg.OIDC.Clients))

		logger.Infof("OIDC provider initialized (issuer=%s)", cfg.OIDC.Issuer)
	} else {
		logger.Infof("OIDC not configured (no key files), authentication disabled")
	}

	return []*rest.APIGroupInfo{
		iamv1.NewAPIGroupInfo(iamProvider, oidcProvider),
	}, oidcProvider
}
