package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
)

// NewAPIGroupInfo creates the APIGroupInfo for the IAM module.
func NewAPIGroupInfo(p *iam.RESTStorageProvider) *rest.APIGroupInfo {
	userStorage := iam.NewUserStorage(p.UserStore(), p.UserNamespaceStore())
	nsStorage := iam.NewNamespaceStorage(p.NamespaceStore(), p.UserStore())

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{Name: "users", Storage: userStorage},
			{
				Name:    "namespaces",
				Storage: nsStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "users", Storage: userStorage},
				},
			},
		},
	}
}
