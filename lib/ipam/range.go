package ipam

import (
	"math/big"
	"net"
)

// Range is a contiguous block of IPs that can be allocated atomically.
//
// For CIDR 10.0.0.0/24: 254 addresses usable out of 256 total (minus network and broadcast).
// For /32 and point-to-point /31: all addresses are usable.
type Range struct {
	net  *net.IPNet
	base *big.Int
	max  int

	alloc *AllocationBitmap
}

// NewCIDRRange creates a Range over a net.IPNet.
// It excludes the network base and broadcast addresses for CIDRs with more than 2 addresses.
func NewCIDRRange(cidr *net.IPNet) (*Range, error) {
	base := bigForIP(cidr.IP)
	size := RangeSize(cidr)

	if size == 0 {
		return nil, ErrNotInRange
	}

	// For any CIDR other than /32 or /128 (size <= 2):
	// exclude network address and broadcast address.
	if size > 2 {
		size -= 2
		base = big.NewInt(0).Add(base, big.NewInt(1))
	}

	return &Range{
		net:   cidr,
		base:  base,
		max:   int(size),
		alloc: NewAllocationBitmap(int(size), cidr.String()),
	}, nil
}

// Allocate attempts to reserve the provided IP.
// Returns ErrNotInRange if the IP is not in this range, or ErrAllocated if already reserved.
func (r *Range) Allocate(ip net.IP) error {
	ok, offset := r.contains(ip)
	if !ok {
		return ErrNotInRange
	}

	if !r.alloc.Allocate(offset) {
		return ErrAllocated
	}
	return nil
}

// AllocateNext reserves one of the IPs from the pool.
// Returns ErrFull if there are no addresses left.
func (r *Range) AllocateNext() (net.IP, error) {
	offset, ok := r.alloc.AllocateNext()
	if !ok {
		return nil, ErrFull
	}
	return addIPOffset(r.base, offset), nil
}

// Release releases the IP back to the pool. Releasing an unallocated IP
// or an IP out of range is a no-op.
func (r *Range) Release(ip net.IP) {
	ok, offset := r.contains(ip)
	if ok {
		r.alloc.Release(offset)
	}
}

// Has returns true if the provided IP is already allocated.
func (r *Range) Has(ip net.IP) bool {
	ok, offset := r.contains(ip)
	if !ok {
		return false
	}
	return r.alloc.Has(offset)
}

// Free returns the count of IP addresses left in the range.
func (r *Range) Free() int {
	return r.alloc.Free()
}

// Used returns the count of IP addresses used in the range.
func (r *Range) Used() int {
	return r.max - r.alloc.Free()
}

// CIDR returns the CIDR covered by the range.
func (r *Range) CIDR() *net.IPNet {
	return r.net
}

// ForEach calls the provided function for each allocated IP.
func (r *Range) ForEach(fn func(net.IP)) {
	r.alloc.ForEach(func(offset int) {
		ip, _ := GetIndexedIP(r.net, offset+1) // +1 because Range skips network address
		fn(ip)
	})
}

// SaveToBytes serializes the bitmap state to []byte for database persistence.
func (r *Range) SaveToBytes() []byte {
	_, data := r.alloc.Snapshot()
	return data
}

// LoadFromBytes restores the bitmap state from []byte read from the database.
func (r *Range) LoadFromBytes(data []byte) error {
	return r.alloc.Restore(r.net.String(), data)
}

// contains returns true and the offset if the ip is in the usable range.
func (r *Range) contains(ip net.IP) (bool, int) {
	if !r.net.Contains(ip) {
		return false, 0
	}

	offset := calculateIPOffset(r.base, ip)
	if offset < 0 || offset >= r.max {
		return false, 0
	}
	return true, offset
}
