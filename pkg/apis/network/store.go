package network

import (
	"context"

	"github.com/jackc/pgx/v5"

	"lcp.io/lcp/pkg/db"
)

// NetworkStore defines database operations on networks.
type NetworkStore interface {
	Create(ctx context.Context, network *DBNetwork) (*DBNetwork, error)
	GetByID(ctx context.Context, id int64) (*DBNetworkWithCount, error)
	Update(ctx context.Context, network *DBNetwork) (*DBNetwork, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBNetwork, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNetworkListRow], error)
	CountSubnets(ctx context.Context, networkID int64) (int64, error)
}

// SubnetStore defines database operations on subnets.
type SubnetStore interface {
	Create(ctx context.Context, tx pgx.Tx, subnet *DBSubnet) (*DBSubnet, error)
	GetByID(ctx context.Context, id int64) (*DBSubnet, error)
	GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*DBSubnet, error)
	Update(ctx context.Context, subnet *DBSubnet) (*DBSubnet, error)
	Patch(ctx context.Context, id int64, fields map[string]any) (*DBSubnet, error)
	UpdateBitmap(ctx context.Context, tx pgx.Tx, id int64, bitmap []byte) error
	UpdateGateway(ctx context.Context, tx pgx.Tx, id int64, gateway string) error
	Delete(ctx context.Context, id int64) error
	DeleteTx(ctx context.Context, tx pgx.Tx, id int64) error
	DeleteByIDs(ctx context.Context, networkID int64, ids []int64) (int64, error)
	List(ctx context.Context, networkID int64, query db.ListQuery) (*db.ListResult[DBSubnet], error)
	ListCIDRsByNetworkID(ctx context.Context, networkID int64) ([]DBSubnetCIDR, error)
	CountNonGatewayAllocations(ctx context.Context, subnetID int64) (int64, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

// IPAllocationStore defines database operations on IP allocations.
type IPAllocationStore interface {
	Create(ctx context.Context, tx pgx.Tx, alloc *DBIPAllocation) (*DBIPAllocation, error)
	GetBySubnetAndIP(ctx context.Context, subnetID int64, ip string) (*DBIPAllocation, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySubnetID(ctx context.Context, tx pgx.Tx, subnetID int64) error
	List(ctx context.Context, subnetID int64, query db.ListQuery) (*db.ListResult[DBIPAllocation], error)
}
