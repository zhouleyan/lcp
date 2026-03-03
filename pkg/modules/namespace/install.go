package namespace

import (
	"lcp.io/lcp/lib/rest"

	nsstore "lcp.io/lcp/pkg/modules/namespace/store"
)

// NewAPIGroupInfo creates the APIGroupInfo for the Namespace module.
func NewAPIGroupInfo(nsStore nsstore.NamespaceStore, unStore nsstore.UserNamespaceStore, userLookup UserLookup) *rest.APIGroupInfo {
	svc := NewNamespaceService(nsStore, unStore, userLookup)
	nsStor := newNamespaceStorage(svc)
	memStor := newMemberStorage(svc)

	return &rest.APIGroupInfo{
		Version: "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "namespaces",
				Storage: nsStor,
				SubResources: []rest.SubResourceInfo{
					{
						Name:    "members",
						Storage: memStor,
					},
				},
			},
		},
	}
}
