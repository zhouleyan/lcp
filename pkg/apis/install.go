package apis

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	iamv1 "lcp.io/lcp/pkg/apis/iam/v1"
	"lcp.io/lcp/pkg/db"
)

// NewAPIGroupInfos creates all APIGroupInfo instances for the server.
// *db.DB is consumed here; downstream packages only see Store interfaces.
func NewAPIGroupInfos(database *db.DB) []*rest.APIGroupInfo {
	iamProvider := iam.NewRESTStorageProvider(
		iamstore.NewPGUserStore(database.Queries),
		iamstore.NewPGWorkspaceStore(database.Pool, database.Queries),
		iamstore.NewPGNamespaceStore(database.Pool, database.Queries),
		iamstore.NewPGUserWorkspaceStore(database.Queries),
		iamstore.NewPGUserNamespaceStore(database.Pool, database.Queries),
	)
	return []*rest.APIGroupInfo{
		iamv1.NewAPIGroupInfo(iamProvider),
	}
}
