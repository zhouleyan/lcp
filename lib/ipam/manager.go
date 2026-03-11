package ipam

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
)

// managedPool holds a CIDRPool along with owner tracking.
type managedPool struct {
	mu     sync.Mutex
	pool   *CIDRPool
	owners map[string]string // ip.String() → owner
}

// Manager provides named pool management with IP allocation, owner tracking, and store persistence.
type Manager struct {
	mu    sync.RWMutex
	pools map[string]*managedPool
	store Store
}

// NewManager creates a new Manager with the given Store.
// Use NoopStore{} for pure in-memory mode.
func NewManager(store Store) *Manager {
	return &Manager{
		pools: make(map[string]*managedPool),
		store: store,
	}
}

// CreatePool creates a new named pool with the given CIDRs.
func (m *Manager) CreatePool(name string, cidrs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pools[name]; exists {
		return ErrPoolExists
	}

	pool := NewCIDRPool()
	for _, cidrStr := range cidrs {
		_, cidr, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("invalid CIDR %q: %w", cidrStr, err)
		}
		if err := pool.AddCIDR(cidr); err != nil {
			return err
		}
	}

	mp := &managedPool{
		pool:   pool,
		owners: make(map[string]string),
	}

	if err := m.store.SavePool(context.Background(), &PoolState{
		Name:  name,
		CIDRs: pool.CIDRs(),
	}); err != nil {
		return fmt.Errorf("persisting pool: %w", err)
	}

	m.pools[name] = mp
	return nil
}

// DeletePool removes a named pool. Returns ErrPoolInUse if it has allocated addresses.
func (m *Manager) DeletePool(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, ok := m.pools[name]
	if !ok {
		return ErrPoolNotFound
	}
	if mp.pool.Used() > 0 {
		return ErrPoolInUse
	}

	if err := m.store.DeletePool(context.Background(), name); err != nil {
		return fmt.Errorf("deleting pool from store: %w", err)
	}

	delete(m.pools, name)
	return nil
}

// GetPool returns info about a named pool.
func (m *Manager) GetPool(name string) (*PoolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.pools[name]
	if !ok {
		return nil, ErrPoolNotFound
	}

	return &PoolInfo{
		Name:     name,
		CIDRs:    mp.pool.CIDRs(),
		UsedIPs:  mp.pool.Used(),
		FreeIPs:  mp.pool.Free(),
		TotalIPs: mp.pool.Used() + mp.pool.Free(),
	}, nil
}

// ListPools returns info for all pools, sorted by name.
func (m *Manager) ListPools() []*PoolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*PoolInfo, 0, len(m.pools))
	for name, mp := range m.pools {
		result = append(result, &PoolInfo{
			Name:     name,
			CIDRs:    mp.pool.CIDRs(),
			UsedIPs:  mp.pool.Used(),
			FreeIPs:  mp.pool.Free(),
			TotalIPs: mp.pool.Used() + mp.pool.Free(),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// AddCIDR adds a CIDR to an existing pool.
func (m *Manager) AddCIDR(pool, cidr string) error {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return ErrPoolNotFound
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	if err := mp.pool.AddCIDR(ipNet); err != nil {
		return err
	}

	if err := m.store.SavePool(context.Background(), &PoolState{
		Name:  pool,
		CIDRs: mp.pool.CIDRs(),
	}); err != nil {
		// Rollback: remove the CIDR we just added
		mp.pool.RemoveCIDR(ipNet)
		return fmt.Errorf("persisting pool: %w", err)
	}

	return nil
}

// RemoveCIDR removes a CIDR from an existing pool.
func (m *Manager) RemoveCIDR(pool, cidr string) error {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return ErrPoolNotFound
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	if err := mp.pool.RemoveCIDR(ipNet); err != nil {
		return err
	}

	if err := m.store.SavePool(context.Background(), &PoolState{
		Name:  pool,
		CIDRs: mp.pool.CIDRs(),
	}); err != nil {
		// Rollback: re-add the CIDR
		mp.pool.AddCIDR(ipNet)
		return fmt.Errorf("persisting pool: %w", err)
	}

	return nil
}

// Allocate reserves a specific IP in the named pool with an owner.
func (m *Manager) Allocate(pool string, ip net.IP, owner string) (*Allocation, error) {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrPoolNotFound
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	if err := mp.pool.Allocate(ip); err != nil {
		return nil, err
	}

	alloc := &Allocation{
		IP:    ip,
		CIDR:  m.findCIDRForIP(mp, ip),
		Owner: owner,
		Pool:  pool,
	}

	if err := m.store.SaveAllocation(context.Background(), alloc); err != nil {
		mp.pool.Release(ip)
		return nil, fmt.Errorf("persisting allocation: %w", err)
	}

	mp.owners[ip.String()] = owner
	return alloc, nil
}

// AllocateNext reserves the next available IP in the named pool with an owner.
func (m *Manager) AllocateNext(pool, owner string) (*Allocation, error) {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrPoolNotFound
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	ip, err := mp.pool.AllocateNext()
	if err != nil {
		return nil, err
	}

	alloc := &Allocation{
		IP:    ip,
		CIDR:  m.findCIDRForIP(mp, ip),
		Owner: owner,
		Pool:  pool,
	}

	if err := m.store.SaveAllocation(context.Background(), alloc); err != nil {
		mp.pool.Release(ip)
		return nil, fmt.Errorf("persisting allocation: %w", err)
	}

	mp.owners[ip.String()] = owner
	return alloc, nil
}

// Release releases an IP back to the named pool.
func (m *Manager) Release(pool string, ip net.IP) error {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return ErrPoolNotFound
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	ipStr := ip.String()
	if _, exists := mp.owners[ipStr]; !exists {
		return ErrNotAllocated
	}

	if err := m.store.DeleteAllocation(context.Background(), pool, ip); err != nil {
		return fmt.Errorf("deleting allocation from store: %w", err)
	}

	mp.pool.Release(ip)
	delete(mp.owners, ipStr)
	return nil
}

// GetAllocation returns the allocation for a specific IP in a pool.
func (m *Manager) GetAllocation(pool string, ip net.IP) (*Allocation, error) {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrPoolNotFound
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	ipStr := ip.String()
	owner, exists := mp.owners[ipStr]
	if !exists {
		return nil, ErrNotAllocated
	}

	return &Allocation{
		IP:    ip,
		CIDR:  m.findCIDRForIP(mp, ip),
		Owner: owner,
		Pool:  pool,
	}, nil
}

// ListAllocations returns all allocations in a named pool.
func (m *Manager) ListAllocations(pool string) ([]*Allocation, error) {
	m.mu.RLock()
	mp, ok := m.pools[pool]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrPoolNotFound
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	result := make([]*Allocation, 0, len(mp.owners))
	for ipStr, owner := range mp.owners {
		ip := net.ParseIP(ipStr)
		result = append(result, &Allocation{
			IP:    ip,
			CIDR:  m.findCIDRForIP(mp, ip),
			Owner: owner,
			Pool:  pool,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].IP.String() < result[j].IP.String()
	})
	return result, nil
}

// ListAllocationsByOwner returns all allocations across all pools for a given owner.
func (m *Manager) ListAllocationsByOwner(owner string) []*Allocation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Allocation
	for poolName, mp := range m.pools {
		mp.mu.Lock()
		for ipStr, o := range mp.owners {
			if o == owner {
				ip := net.ParseIP(ipStr)
				result = append(result, &Allocation{
					IP:    ip,
					CIDR:  m.findCIDRForIP(mp, ip),
					Owner: owner,
					Pool:  poolName,
				})
			}
		}
		mp.mu.Unlock()
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Pool != result[j].Pool {
			return result[i].Pool < result[j].Pool
		}
		return result[i].IP.String() < result[j].IP.String()
	})
	return result
}

// findCIDRForIP returns the CIDR string that contains the given IP.
func (m *Manager) findCIDRForIP(mp *managedPool, ip net.IP) string {
	for _, cidrStr := range mp.pool.CIDRs() {
		_, cidr, _ := net.ParseCIDR(cidrStr)
		if cidr.Contains(ip) {
			return cidrStr
		}
	}
	return ""
}
