package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
)

// NewAPIGroupInfo creates the APIGroupInfo for the IAM module.
func NewAPIGroupInfo(p *iam.RESTStorageProvider) *rest.APIGroupInfo {
	userStorage := iam.NewUserStorage(p.UserStore())
	wsStorage := iam.NewWorkspaceStorage(p.WorkspaceStore(), p.UserStore())
	nsStorage := iam.NewNamespaceStorage(p.NamespaceStore(), p.WorkspaceStore(), p.UserStore())
	wsUserStorage := iam.NewWorkspaceUserStorage(p.UserWorkspaceStore(), p.UserStore())
	nsUserStorage := iam.NewNamespaceUserStorage(p.UserNamespaceStore(), p.UserStore())

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{Name: "users", Storage: userStorage},
			{
				Name:    "workspaces",
				Storage: wsStorage,
				SubResources: []rest.ResourceInfo{
					{
						Name:    "namespaces",
						Storage: nsStorage,
						SubResources: []rest.ResourceInfo{
							{Name: "users", Storage: nsUserStorage},
						},
					},
					{Name: "users", Storage: wsUserStorage},
				},
			},
			{
				Name:    "namespaces",
				Storage: nsStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "users", Storage: nsUserStorage},
				},
			},
		},
	}
}
