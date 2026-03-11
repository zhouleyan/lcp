package ipam

import (
	"testing"
)

func TestAllocationBitmap_AllocateAndRelease(t *testing.T) {
	b := NewAllocationBitmap(10, "test-range")

	if !b.Allocate(3) {
		t.Fatal("expected allocation of offset 3 to succeed")
	}
	if b.Allocate(3) {
		t.Fatal("expected duplicate allocation of offset 3 to fail")
	}
	if !b.Has(3) {
		t.Fatal("expected Has(3) to return true")
	}
	if b.Free() != 9 {
		t.Fatalf("expected 9 free, got %d", b.Free())
	}

	b.Release(3)
	if b.Has(3) {
		t.Fatal("expected Has(3) to return false after release")
	}
	if b.Free() != 10 {
		t.Fatalf("expected 10 free, got %d", b.Free())
	}
}

func TestAllocationBitmap_AllocateNext(t *testing.T) {
	b := NewAllocationBitmap(5, "test-range")

	allocated := make(map[int]bool)
	for range 5 {
		offset, ok := b.AllocateNext()
		if !ok {
			t.Fatal("expected AllocateNext to succeed")
		}
		if allocated[offset] {
			t.Fatalf("offset %d allocated twice", offset)
		}
		allocated[offset] = true
	}

	if _, ok := b.AllocateNext(); ok {
		t.Fatal("expected AllocateNext to fail when full")
	}
	if b.Free() != 0 {
		t.Fatalf("expected 0 free, got %d", b.Free())
	}
}

func TestAllocationBitmap_ContiguousStrategy(t *testing.T) {
	b := NewContiguousAllocationBitmap(5, "test-range")

	for i := range 5 {
		offset, ok := b.AllocateNext()
		if !ok {
			t.Fatal("expected AllocateNext to succeed")
		}
		if offset != i {
			t.Fatalf("expected contiguous offset %d, got %d", i, offset)
		}
	}
}

func TestAllocationBitmap_ForEach(t *testing.T) {
	b := NewContiguousAllocationBitmap(100, "test-range")

	b.Allocate(5)
	b.Allocate(10)
	b.Allocate(50)

	var collected []int
	b.ForEach(func(offset int) {
		collected = append(collected, offset)
	})

	if len(collected) != 3 {
		t.Fatalf("expected 3 allocated offsets, got %d", len(collected))
	}
	expected := map[int]bool{5: true, 10: true, 50: true}
	for _, v := range collected {
		if !expected[v] {
			t.Fatalf("unexpected offset %d in ForEach", v)
		}
	}
}

func TestAllocationBitmap_SnapshotRestore(t *testing.T) {
	b := NewAllocationBitmap(100, "test-range")
	b.Allocate(1)
	b.Allocate(50)
	b.Allocate(99)

	rangeSpec, data := b.Snapshot()

	b2 := NewAllocationBitmap(100, "test-range")
	if err := b2.Restore(rangeSpec, data); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	if b2.Free() != 97 {
		t.Fatalf("expected 97 free after restore, got %d", b2.Free())
	}
	for _, offset := range []int{1, 50, 99} {
		if !b2.Has(offset) {
			t.Fatalf("expected offset %d to be allocated after restore", offset)
		}
	}
}

func TestAllocationBitmap_RestoreMismatch(t *testing.T) {
	b := NewAllocationBitmap(10, "range-a")
	if err := b.Restore("range-b", nil); err == nil {
		t.Fatal("expected error when restoring with mismatched range spec")
	}
}

func TestAllocationBitmap_ReleaseNoOp(t *testing.T) {
	b := NewAllocationBitmap(10, "test-range")
	// Releasing an unallocated offset should be a no-op
	b.Release(5)
	if b.Free() != 10 {
		t.Fatalf("expected 10 free after no-op release, got %d", b.Free())
	}
}

func TestAllocationBitmap_FullAllocateReleaseCycle(t *testing.T) {
	const size = 254
	b := NewAllocationBitmap(size, "test-range")

	// Allocate all
	for i := range size {
		if !b.Allocate(i) {
			t.Fatalf("expected allocation of offset %d to succeed", i)
		}
	}
	if b.Free() != 0 {
		t.Fatalf("expected 0 free, got %d", b.Free())
	}

	// Release all
	for i := range size {
		b.Release(i)
	}
	if b.Free() != size {
		t.Fatalf("expected %d free, got %d", size, b.Free())
	}

	// Re-allocate via AllocateNext
	for range size {
		if _, ok := b.AllocateNext(); !ok {
			t.Fatal("expected AllocateNext to succeed after full release")
		}
	}
	if b.Free() != 0 {
		t.Fatal("expected 0 free after full re-allocation")
	}
}
