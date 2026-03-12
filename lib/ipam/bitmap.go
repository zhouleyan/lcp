package ipam

import (
	"errors"
	"math/big"
	"math/bits"
	"math/rand/v2"
	"sync"
)

// AllocationBitmap is a contiguous block of resources that can be allocated atomically.
//
// Each resource has an offset. The internal structure is a bitmap, with a bit for each offset.
// If a resource is taken, the bit at that offset is set to one.
type AllocationBitmap struct {
	strategy  bitAllocator
	max       int
	rangeSpec string

	mu        sync.Mutex
	count     int
	allocated *big.Int
}

// bitAllocator represents a search strategy in the allocation map for a valid item.
type bitAllocator interface {
	AllocateBit(allocated *big.Int, max, count int) (int, bool)
}

// NewAllocationBitmap creates an allocation bitmap using the random scan strategy.
func NewAllocationBitmap(max int, rangeSpec string) *AllocationBitmap {
	return &AllocationBitmap{
		strategy:  randomScanStrategy{},
		allocated: big.NewInt(0),
		max:       max,
		rangeSpec: rangeSpec,
	}
}

// NewContiguousAllocationBitmap creates an allocation bitmap using the contiguous scan strategy.
func NewContiguousAllocationBitmap(max int, rangeSpec string) *AllocationBitmap {
	return &AllocationBitmap{
		strategy:  contiguousScanStrategy{},
		allocated: big.NewInt(0),
		max:       max,
		rangeSpec: rangeSpec,
	}
}

// Allocate attempts to reserve the provided offset.
// Returns true if it was allocated, false if it was already in use.
func (b *AllocationBitmap) Allocate(offset int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.allocated.Bit(offset) == 1 {
		return false
	}
	b.allocated = b.allocated.SetBit(b.allocated, offset, 1)
	b.count++
	return true
}

// AllocateNext reserves one of the items from the pool.
// Returns (0, false) if there are no items left.
func (b *AllocationBitmap) AllocateNext() (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	next, ok := b.strategy.AllocateBit(b.allocated, b.max, b.count)
	if !ok {
		return 0, false
	}
	b.count++
	b.allocated = b.allocated.SetBit(b.allocated, next, 1)
	return next, true
}

// NextFree returns the smallest unallocated offset without reserving it.
// Returns (0, false) if all items are allocated.
func (b *AllocationBitmap) NextFree() (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count >= b.max {
		return 0, false
	}
	for i := range b.max {
		if b.allocated.Bit(i) == 0 {
			return i, true
		}
	}
	return 0, false
}

// Release releases the item back to the pool. Releasing an
// unallocated item or an item out of the range is a no-op.
func (b *AllocationBitmap) Release(offset int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.allocated.Bit(offset) == 0 {
		return
	}
	b.allocated = b.allocated.SetBit(b.allocated, offset, 0)
	b.count--
}

// Has returns true if the provided offset is already allocated.
func (b *AllocationBitmap) Has(offset int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.allocated.Bit(offset) == 1
}

// Free returns the count of items left in the range.
func (b *AllocationBitmap) Free() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.max - b.count
}

const (
	notZero   = uint64(^big.Word(0))
	wordPower = (notZero>>8)&1 + (notZero>>16)&1 + (notZero>>32)&1
	wordSize  = 1 << wordPower
)

// ForEach calls the provided function for each allocated bit.
func (b *AllocationBitmap) ForEach(fn func(int)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	words := b.allocated.Bits()
	for wordIdx, word := range words {
		bit := 0
		for word > 0 {
			if (word & 1) != 0 {
				fn((wordIdx * wordSize * 8) + bit)
				word = word &^ 1
			}
			bit++
			word = word >> 1
		}
	}
}

// Snapshot saves the current state of the pool.
func (b *AllocationBitmap) Snapshot() (string, []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.rangeSpec, b.allocated.Bytes()
}

// Restore restores the pool to the previously captured state.
func (b *AllocationBitmap) Restore(rangeSpec string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rangeSpec != rangeSpec {
		return errors.New("the provided range does not match the current range")
	}

	b.allocated = big.NewInt(0).SetBytes(data)
	b.count = countBits(b.allocated)
	return nil
}

// countBits returns the number of set bits in n.
func countBits(n *big.Int) int {
	var count int
	for _, w := range n.Bits() {
		count += bits.OnesCount64(uint64(w))
	}
	return count
}

// randomScanStrategy chooses a random address from the provided big.Int, and then
// scans forward looking for the next available address (wrapping if necessary).
type randomScanStrategy struct{}

func (randomScanStrategy) AllocateBit(allocated *big.Int, max, count int) (int, bool) {
	if count >= max {
		return 0, false
	}
	offset := rand.IntN(max)
	for i := range max {
		at := (offset + i) % max
		if allocated.Bit(at) == 0 {
			return at, true
		}
	}
	return 0, false
}

// contiguousScanStrategy tries to allocate starting at 0 and filling in any gaps.
type contiguousScanStrategy struct{}

func (contiguousScanStrategy) AllocateBit(allocated *big.Int, max, count int) (int, bool) {
	if count >= max {
		return 0, false
	}
	for i := range max {
		if allocated.Bit(i) == 0 {
			return i, true
		}
	}
	return 0, false
}
