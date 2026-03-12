package v1

import (
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/network"
	networkstore "lcp.io/lcp/pkg/apis/network/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of Network module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewNetworkModule 初始化 Network 模块并构建 API group。
func NewNetworkModule(database *db.DB) ModuleResult {
	p := network.NewRESTStorageProvider(networkstore.NewStores(database))

	networkStorage := network.NewNetworkStorage(p.Network)
	subnetStorage := network.NewSubnetStorage(p.Subnet, p.IPAllocation, p.Network)
	allocationStorage := network.NewAllocationStorage(p.IPAllocation, p.Subnet)

	group := &rest.APIGroupInfo{
		GroupName: "network",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "networks",
				Storage: networkStorage,
				SubResources: []rest.ResourceInfo{
					{
						Name:    "subnets",
						Storage: subnetStorage,
						SubResources: []rest.ResourceInfo{
							{
								Name:    "allocations",
								Storage: allocationStorage,
							},
						},
					},
				},
			},
		},
	}

	return ModuleResult{Group: group}
}
