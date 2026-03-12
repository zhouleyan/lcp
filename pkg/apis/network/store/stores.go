package store

import (
	"lcp.io/lcp/pkg/apis/network"
	"lcp.io/lcp/pkg/db"
)

// NewStores 创建所有 network store 实现。
func NewStores(database *db.DB) network.Stores {
	return network.Stores{
		Network:      NewPGNetworkStore(database.Pool, database.Queries),
		Subnet:       NewPGSubnetStore(database.Pool, database.Queries),
		IPAllocation: NewPGIPAllocationStore(database.Pool, database.Queries),
	}
}
