package ipam

import (
	"context"
	"net"
)

// Store defines the persistence interface for IPAM state.
type Store interface {
	SavePool(ctx context.Context, state *PoolState) error
	DeletePool(ctx context.Context, name string) error
	LoadPools(ctx context.Context) ([]*PoolState, error)
	SaveAllocation(ctx context.Context, alloc *Allocation) error
	DeleteAllocation(ctx context.Context, pool string, ip net.IP) error
	LoadAllocations(ctx context.Context, pool string) ([]*Allocation, error)
}

// PoolState captures the persistent state of a named pool.
type PoolState struct {
	Name  string
	CIDRs []string
}

// NoopStore is a Store implementation that does nothing.
// All methods return nil, providing a pure in-memory mode.
type NoopStore struct{}

func (NoopStore) SavePool(context.Context, *PoolState) error                  { return nil }
func (NoopStore) DeletePool(context.Context, string) error                    { return nil }
func (NoopStore) LoadPools(context.Context) ([]*PoolState, error)             { return nil, nil }
func (NoopStore) SaveAllocation(context.Context, *Allocation) error           { return nil }
func (NoopStore) DeleteAllocation(context.Context, string, net.IP) error      { return nil }
func (NoopStore) LoadAllocations(context.Context, string) ([]*Allocation, error) { return nil, nil }
