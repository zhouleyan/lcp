package user

import (
	"lcp.io/lcp/lib/rest"

	userstore "lcp.io/lcp/pkg/modules/user/store"
)

// NewAPIGroupInfo creates the APIGroupInfo for the User module.
func NewAPIGroupInfo(s userstore.UserStore) *rest.APIGroupInfo {
	svc := NewUserService(s)
	storage := newUserStorage(svc)

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "users",
				Storage: storage,
			},
		},
	}
}
