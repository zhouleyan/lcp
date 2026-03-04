package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	"lcp.io/lcp/pkg/db"
)

// NewAPIGroupInfo creates the APIGroupInfo for the IAM module.
func NewAPIGroupInfo(database *db.DB) *rest.APIGroupInfo {
	// Create stores
	userStore := iamstore.NewPGUserStore(database.Pool, database.Queries)
	nsStore := iamstore.NewPGNamespaceStore(database.Pool, database.Queries)
	unStore := iamstore.NewPGUserNamespaceStore(database.Queries)

	// Create storage layers
	userStor := iam.NewUserStorage(userStore)
	nsStor := iam.NewNamespaceStorage(nsStore, userStore)
	memStor := iam.NewMemberStorage(nsStore, unStore, userStore)

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{Name: "users", Storage: userStor},
			{
				Name:    "namespaces",
				Storage: nsStor,
				SubResources: []rest.ResourceInfo{
					{Name: "members", Storage: memStor},
				},
			},
		},
	}
}
