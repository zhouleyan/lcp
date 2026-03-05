package v1

import (
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
)

// NewAPIGroupInfo creates the APIGroupInfo for the IAM module.
func NewAPIGroupInfo(p *iam.RESTStorageProvider, provider *oidc.Provider) *rest.APIGroupInfo {
	var userStorage rest.StandardStorage
	var changePasswordActions []rest.ActionInfo

	if provider != nil {
		ps := provider.PasswordService()
		userStorage = iam.NewUserStorageWithPassword(p.UserStore(), ps.Hash)
		changePasswordActions = []rest.ActionInfo{
			{
				Name:   "change-password",
				Method: "POST",
				Handler: iam.NewChangePasswordHandler(
					p.UserStore(),
					p.RefreshTokenStore(),
					ps.Hash,
					ps.Verify,
				),
			},
		}
	} else {
		userStorage = iam.NewUserStorage(p.UserStore())
	}

	wsStorage := iam.NewWorkspaceStorage(p.WorkspaceStore(), p.UserStore())
	nsStorage := iam.NewNamespaceStorage(p.NamespaceStore(), p.WorkspaceStore(), p.UserStore())
	wsUserStorage := iam.NewWorkspaceUserStorage(p.UserWorkspaceStore(), p.UserStore())
	nsUserStorage := iam.NewNamespaceUserStorage(p.UserNamespaceStore(), p.UserStore())

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{Name: "users", Storage: userStorage, Actions: changePasswordActions},
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
