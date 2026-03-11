package ipam

import (
	"net"
	"sync"
	"testing"
)

func TestManager_CreateDeletePool(t *testing.T) {
	m := NewManager(NoopStore{})

	if err := m.CreatePool("vpc-prod", []string{"10.0.0.0/24"}); err != nil {
		t.Fatalf("CreatePool: %v", err)
	}
	if err := m.CreatePool("vpc-prod", []string{"10.0.1.0/24"}); err != ErrPoolExists {
		t.Fatalf("expected ErrPoolExists, got %v", err)
	}

	info, err := m.GetPool("vpc-prod")
	if err != nil {
		t.Fatalf("GetPool: %v", err)
	}
	if info.TotalIPs != 254 {
		t.Fatalf("expected 254 total IPs, got %d", info.TotalIPs)
	}

	if err := m.DeletePool("vpc-prod"); err != nil {
		t.Fatalf("DeletePool: %v", err)
	}
	if _, err := m.GetPool("vpc-prod"); err != ErrPoolNotFound {
		t.Fatalf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestManager_DeletePoolInUse(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})
	m.AllocateNext("test", "host-1")

	if err := m.DeletePool("test"); err != ErrPoolInUse {
		t.Fatalf("expected ErrPoolInUse, got %v", err)
	}
}

func TestManager_AllocateSpecific(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})

	alloc, err := m.Allocate("test", net.ParseIP("10.0.0.1"), "host-web-01")
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if alloc.Owner != "host-web-01" {
		t.Fatalf("expected owner host-web-01, got %s", alloc.Owner)
	}
	if alloc.Pool != "test" {
		t.Fatalf("expected pool test, got %s", alloc.Pool)
	}

	// Duplicate
	if _, err := m.Allocate("test", net.ParseIP("10.0.0.1"), "host-web-02"); err != ErrAllocated {
		t.Fatalf("expected ErrAllocated, got %v", err)
	}
}

func TestManager_AllocateNext(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/30"}) // 2 usable IPs

	a1, err := m.AllocateNext("test", "host-1")
	if err != nil {
		t.Fatalf("AllocateNext: %v", err)
	}
	a2, err := m.AllocateNext("test", "host-2")
	if err != nil {
		t.Fatalf("AllocateNext: %v", err)
	}
	if a1.IP.Equal(a2.IP) {
		t.Fatal("expected different IPs")
	}

	if _, err := m.AllocateNext("test", "host-3"); err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestManager_Release(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})

	alloc, _ := m.AllocateNext("test", "host-1")

	if err := m.Release("test", alloc.IP); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Release again should fail
	if err := m.Release("test", alloc.IP); err != ErrNotAllocated {
		t.Fatalf("expected ErrNotAllocated, got %v", err)
	}
}

func TestManager_GetAllocation(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})

	m.Allocate("test", net.ParseIP("10.0.0.5"), "host-5")

	alloc, err := m.GetAllocation("test", net.ParseIP("10.0.0.5"))
	if err != nil {
		t.Fatalf("GetAllocation: %v", err)
	}
	if alloc.Owner != "host-5" {
		t.Fatalf("expected owner host-5, got %s", alloc.Owner)
	}

	// Not allocated
	if _, err := m.GetAllocation("test", net.ParseIP("10.0.0.99")); err != ErrNotAllocated {
		t.Fatalf("expected ErrNotAllocated, got %v", err)
	}
}

func TestManager_ListAllocations(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})

	m.Allocate("test", net.ParseIP("10.0.0.1"), "host-1")
	m.Allocate("test", net.ParseIP("10.0.0.2"), "host-2")

	allocs, err := m.ListAllocations("test")
	if err != nil {
		t.Fatalf("ListAllocations: %v", err)
	}
	if len(allocs) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(allocs))
	}
}

func TestManager_ListAllocationsByOwner(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("vpc-a", []string{"10.0.0.0/24"})
	m.CreatePool("vpc-b", []string{"10.0.1.0/24"})

	m.Allocate("vpc-a", net.ParseIP("10.0.0.1"), "host-shared")
	m.Allocate("vpc-b", net.ParseIP("10.0.1.1"), "host-shared")
	m.Allocate("vpc-a", net.ParseIP("10.0.0.2"), "host-other")

	allocs := m.ListAllocationsByOwner("host-shared")
	if len(allocs) != 2 {
		t.Fatalf("expected 2 allocations for host-shared, got %d", len(allocs))
	}
	for _, a := range allocs {
		if a.Owner != "host-shared" {
			t.Fatalf("expected owner host-shared, got %s", a.Owner)
		}
	}
}

func TestManager_ListPools(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("b-pool", []string{"10.0.0.0/24"})
	m.CreatePool("a-pool", []string{"10.0.1.0/24"})

	pools := m.ListPools()
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	// Should be sorted by name
	if pools[0].Name != "a-pool" || pools[1].Name != "b-pool" {
		t.Fatalf("expected sorted pools [a-pool, b-pool], got [%s, %s]", pools[0].Name, pools[1].Name)
	}
}

func TestManager_AddRemoveCIDR(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"})

	if err := m.AddCIDR("test", "10.0.1.0/24"); err != nil {
		t.Fatalf("AddCIDR: %v", err)
	}

	info, _ := m.GetPool("test")
	if len(info.CIDRs) != 2 {
		t.Fatalf("expected 2 CIDRs, got %d", len(info.CIDRs))
	}

	if err := m.RemoveCIDR("test", "10.0.1.0/24"); err != nil {
		t.Fatalf("RemoveCIDR: %v", err)
	}

	info, _ = m.GetPool("test")
	if len(info.CIDRs) != 1 {
		t.Fatalf("expected 1 CIDR, got %d", len(info.CIDRs))
	}
}

func TestManager_PoolNotFound(t *testing.T) {
	m := NewManager(NoopStore{})

	if _, err := m.Allocate("nonexistent", net.ParseIP("10.0.0.1"), "x"); err != ErrPoolNotFound {
		t.Fatalf("expected ErrPoolNotFound, got %v", err)
	}
	if _, err := m.AllocateNext("nonexistent", "x"); err != ErrPoolNotFound {
		t.Fatalf("expected ErrPoolNotFound, got %v", err)
	}
	if err := m.Release("nonexistent", net.ParseIP("10.0.0.1")); err != ErrPoolNotFound {
		t.Fatalf("expected ErrPoolNotFound, got %v", err)
	}
	if err := m.DeletePool("nonexistent"); err != ErrPoolNotFound {
		t.Fatalf("expected ErrPoolNotFound, got %v", err)
	}
}

func TestManager_ConcurrentAllocateRelease(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"}) // 254 usable IPs

	const goroutines = 50
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range opsPerGoroutine {
				alloc, err := m.AllocateNext("test", "concurrent-owner")
				if err != nil {
					continue // pool might be full momentarily
				}
				m.Release("test", alloc.IP)
			}
		}()
	}

	wg.Wait()

	info, _ := m.GetPool("test")
	if info.UsedIPs != 0 {
		t.Fatalf("expected 0 used IPs after concurrent alloc/release, got %d", info.UsedIPs)
	}
}

func TestManager_ConcurrentAllocateUnique(t *testing.T) {
	m := NewManager(NoopStore{})
	m.CreatePool("test", []string{"10.0.0.0/24"}) // 254 usable IPs

	const count = 254
	results := make(chan string, count)

	var wg sync.WaitGroup
	wg.Add(count)

	for i := range count {
		go func(idx int) {
			defer wg.Done()
			alloc, err := m.AllocateNext("test", "owner")
			if err != nil {
				t.Errorf("goroutine %d: AllocateNext: %v", idx, err)
				return
			}
			results <- alloc.IP.String()
		}(i)
	}

	wg.Wait()
	close(results)

	seen := make(map[string]bool)
	for ip := range results {
		if seen[ip] {
			t.Fatalf("duplicate IP allocated: %s", ip)
		}
		seen[ip] = true
	}

	if len(seen) != count {
		t.Fatalf("expected %d unique IPs, got %d", count, len(seen))
	}
}

func TestManager_InvalidCIDR(t *testing.T) {
	m := NewManager(NoopStore{})

	if err := m.CreatePool("test", []string{"not-a-cidr"}); err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}
