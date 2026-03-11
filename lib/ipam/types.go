package ipam

import (
	"errors"
	"net"
)

var (
	ErrFull         = errors.New("ipam: all addresses are allocated")
	ErrAllocated    = errors.New("ipam: address is already allocated")
	ErrNotInRange   = errors.New("ipam: address is not in the valid range")
	ErrPoolNotFound = errors.New("ipam: pool not found")
	ErrPoolExists   = errors.New("ipam: pool already exists")
	ErrPoolInUse    = errors.New("ipam: pool has allocated addresses")
	ErrCIDRExists   = errors.New("ipam: CIDR already exists in pool")
	ErrCIDRInUse    = errors.New("ipam: CIDR has allocated addresses")
	ErrNotAllocated = errors.New("ipam: address is not allocated")
)

// Allocation represents a single IP address allocation within a pool.
type Allocation struct {
	IP    net.IP `json:"ip"`
	CIDR  string `json:"cidr"`
	Owner string `json:"owner"`
	Pool  string `json:"pool"`
}

// PoolInfo provides a read-only summary of a named IP pool.
type PoolInfo struct {
	Name     string   `json:"name"`
	CIDRs    []string `json:"cidrs"`
	UsedIPs  int      `json:"usedIPs"`
	FreeIPs  int      `json:"freeIPs"`
	TotalIPs int      `json:"totalIPs"`
}
