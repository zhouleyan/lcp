package ipam

import (
	"fmt"
	"net"
	"sync"
)

// CIDRPool manages IP allocation across multiple CIDRs.
// It maintains one Range allocator per CIDR.
type CIDRPool struct {
	mu         sync.RWMutex
	allocators []*Range
	cidrSet    map[string]struct{} // dedup key is canonical CIDR string
}

// NewCIDRPool creates an empty CIDRPool.
func NewCIDRPool() *CIDRPool {
	return &CIDRPool{
		cidrSet: make(map[string]struct{}),
	}
}

// AddCIDR adds a new CIDR range to the pool.
// Returns ErrCIDRExists if the CIDR is already in the pool.
func (p *CIDRPool) AddCIDR(cidr *net.IPNet) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := cidr.String()
	if _, exists := p.cidrSet[key]; exists {
		return ErrCIDRExists
	}

	r, err := NewCIDRRange(cidr)
	if err != nil {
		return fmt.Errorf("creating range for %s: %w", key, err)
	}

	p.allocators = append(p.allocators, r)
	p.cidrSet[key] = struct{}{}
	return nil
}

// RemoveCIDR removes a CIDR range from the pool.
// Returns ErrCIDRInUse if the CIDR has allocated IPs.
func (p *CIDRPool) RemoveCIDR(cidr *net.IPNet) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := cidr.String()
	if _, exists := p.cidrSet[key]; !exists {
		return ErrNotInRange
	}

	for i, r := range p.allocators {
		if r.CIDR().String() == key {
			if r.Used() > 0 {
				return ErrCIDRInUse
			}
			p.allocators = append(p.allocators[:i], p.allocators[i+1:]...)
			delete(p.cidrSet, key)
			return nil
		}
	}
	return ErrNotInRange
}

// Allocate attempts to reserve the provided IP in the appropriate CIDR range.
func (p *CIDRPool) Allocate(ip net.IP) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, r := range p.allocators {
		if r.CIDR().Contains(ip) {
			return r.Allocate(ip)
		}
	}
	return ErrNotInRange
}

// AllocateNext reserves the next available IP from any CIDR in the pool.
// CIDRs are tried in order; exhausted CIDRs are skipped.
func (p *CIDRPool) AllocateNext() (net.IP, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, r := range p.allocators {
		if r.Free() == 0 {
			continue
		}
		return r.AllocateNext()
	}
	return nil, ErrFull
}

// Release releases the IP back to the appropriate CIDR range.
func (p *CIDRPool) Release(ip net.IP) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, r := range p.allocators {
		if r.CIDR().Contains(ip) {
			r.Release(ip)
			return
		}
	}
}

// Has returns true if the provided IP is allocated in any CIDR.
func (p *CIDRPool) Has(ip net.IP) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, r := range p.allocators {
		if r.Has(ip) {
			return true
		}
	}
	return false
}

// Free returns the total number of free IPs across all CIDRs.
func (p *CIDRPool) Free() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var total int
	for _, r := range p.allocators {
		total += r.Free()
	}
	return total
}

// Used returns the total number of used IPs across all CIDRs.
func (p *CIDRPool) Used() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var total int
	for _, r := range p.allocators {
		total += r.Used()
	}
	return total
}

// CIDRs returns the list of CIDR strings in the pool.
func (p *CIDRPool) CIDRs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]string, 0, len(p.allocators))
	for _, r := range p.allocators {
		result = append(result, r.CIDR().String())
	}
	return result
}

// ForEach calls the provided function for each allocated IP across all CIDRs.
func (p *CIDRPool) ForEach(fn func(net.IP)) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, r := range p.allocators {
		r.ForEach(fn)
	}
}
