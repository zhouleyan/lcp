package ipam

import (
	"net"
	"testing"
)

func TestCIDRPool_AddRemoveCIDR(t *testing.T) {
	p := NewCIDRPool()
	cidr := mustParseCIDR(t, "10.0.0.0/24")

	if err := p.AddCIDR(cidr); err != nil {
		t.Fatalf("AddCIDR: %v", err)
	}
	if err := p.AddCIDR(cidr); err != ErrCIDRExists {
		t.Fatalf("expected ErrCIDRExists, got %v", err)
	}

	cidrs := p.CIDRs()
	if len(cidrs) != 1 || cidrs[0] != "10.0.0.0/24" {
		t.Fatalf("unexpected CIDRs: %v", cidrs)
	}

	if err := p.RemoveCIDR(cidr); err != nil {
		t.Fatalf("RemoveCIDR: %v", err)
	}
	if len(p.CIDRs()) != 0 {
		t.Fatal("expected 0 CIDRs after removal")
	}
}

func TestCIDRPool_RemoveCIDRInUse(t *testing.T) {
	p := NewCIDRPool()
	cidr := mustParseCIDR(t, "10.0.0.0/24")
	p.AddCIDR(cidr)
	p.Allocate(net.ParseIP("10.0.0.1"))

	if err := p.RemoveCIDR(cidr); err != ErrCIDRInUse {
		t.Fatalf("expected ErrCIDRInUse, got %v", err)
	}
}

func TestCIDRPool_AllocateSpecific(t *testing.T) {
	p := NewCIDRPool()
	p.AddCIDR(mustParseCIDR(t, "10.0.0.0/24"))

	if err := p.Allocate(net.ParseIP("10.0.0.1")); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if !p.Has(net.ParseIP("10.0.0.1")) {
		t.Fatal("expected Has to return true")
	}

	// Out of range
	if err := p.Allocate(net.ParseIP("192.168.0.1")); err != ErrNotInRange {
		t.Fatalf("expected ErrNotInRange, got %v", err)
	}
}

func TestCIDRPool_AllocateNextMultipleCIDRs(t *testing.T) {
	p := NewCIDRPool()
	p.AddCIDR(mustParseCIDR(t, "10.0.0.0/30")) // 2 usable IPs
	p.AddCIDR(mustParseCIDR(t, "10.0.1.0/30")) // 2 usable IPs

	if p.Free() != 4 {
		t.Fatalf("expected 4 free, got %d", p.Free())
	}

	ips := make(map[string]bool)
	for range 4 {
		ip, err := p.AllocateNext()
		if err != nil {
			t.Fatalf("AllocateNext: %v", err)
		}
		ips[ip.String()] = true
	}

	if _, err := p.AllocateNext(); err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}

	// All 4 IPs should be from the two CIDRs
	if len(ips) != 4 {
		t.Fatalf("expected 4 unique IPs, got %d", len(ips))
	}
}

func TestCIDRPool_Release(t *testing.T) {
	p := NewCIDRPool()
	p.AddCIDR(mustParseCIDR(t, "10.0.0.0/24"))

	p.Allocate(net.ParseIP("10.0.0.1"))
	if p.Used() != 1 {
		t.Fatalf("expected 1 used, got %d", p.Used())
	}

	p.Release(net.ParseIP("10.0.0.1"))
	if p.Used() != 0 {
		t.Fatalf("expected 0 used after release, got %d", p.Used())
	}

	// Release out-of-range is a no-op
	p.Release(net.ParseIP("192.168.0.1"))
}

func TestCIDRPool_ForEach(t *testing.T) {
	p := NewCIDRPool()
	p.AddCIDR(mustParseCIDR(t, "10.0.0.0/24"))
	p.AddCIDR(mustParseCIDR(t, "10.0.1.0/24"))

	p.Allocate(net.ParseIP("10.0.0.1"))
	p.Allocate(net.ParseIP("10.0.1.1"))

	var collected []string
	p.ForEach(func(ip net.IP) {
		collected = append(collected, ip.String())
	})

	if len(collected) != 2 {
		t.Fatalf("expected 2 IPs in ForEach, got %d", len(collected))
	}
}

func TestCIDRPool_FreeUsedTotals(t *testing.T) {
	p := NewCIDRPool()
	p.AddCIDR(mustParseCIDR(t, "10.0.0.0/30")) // 2 usable
	p.AddCIDR(mustParseCIDR(t, "10.0.1.0/30")) // 2 usable

	if p.Free() != 4 || p.Used() != 0 {
		t.Fatalf("expected 4 free 0 used, got %d free %d used", p.Free(), p.Used())
	}

	p.Allocate(net.ParseIP("10.0.0.1"))
	if p.Free() != 3 || p.Used() != 1 {
		t.Fatalf("expected 3 free 1 used, got %d free %d used", p.Free(), p.Used())
	}
}
