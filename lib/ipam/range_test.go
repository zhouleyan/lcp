package ipam

import (
	"net"
	"testing"
)

func mustParseCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("ParseCIDR(%s): %v", s, err)
	}
	return cidr
}

func TestRange_Basic(t *testing.T) {
	r, err := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/24"))
	if err != nil {
		t.Fatalf("NewCIDRRange: %v", err)
	}

	// /24 = 256 addresses - 2 (network + broadcast) = 254 usable
	if r.Free() != 254 {
		t.Fatalf("expected 254 free, got %d", r.Free())
	}
	if r.Used() != 0 {
		t.Fatalf("expected 0 used, got %d", r.Used())
	}
}

func TestRange_AllocateSpecific(t *testing.T) {
	r, _ := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/24"))

	if err := r.Allocate(net.ParseIP("10.0.0.1")); err != nil {
		t.Fatalf("Allocate(10.0.0.1): %v", err)
	}
	if !r.Has(net.ParseIP("10.0.0.1")) {
		t.Fatal("expected Has(10.0.0.1) to be true")
	}

	// Duplicate
	if err := r.Allocate(net.ParseIP("10.0.0.1")); err != ErrAllocated {
		t.Fatalf("expected ErrAllocated, got %v", err)
	}

	// Out of range
	if err := r.Allocate(net.ParseIP("10.0.1.1")); err != ErrNotInRange {
		t.Fatalf("expected ErrNotInRange, got %v", err)
	}

	// Network address (10.0.0.0) should be excluded
	if err := r.Allocate(net.ParseIP("10.0.0.0")); err != ErrNotInRange {
		t.Fatalf("expected network address to be excluded, got %v", err)
	}

	// Broadcast address (10.0.0.255) should be excluded
	if err := r.Allocate(net.ParseIP("10.0.0.255")); err != ErrNotInRange {
		t.Fatalf("expected broadcast address to be excluded, got %v", err)
	}
}

func TestRange_AllocateNext(t *testing.T) {
	r, _ := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/30"))

	// /30 = 4 addresses - 2 = 2 usable (10.0.0.1, 10.0.0.2)
	if r.Free() != 2 {
		t.Fatalf("expected 2 free, got %d", r.Free())
	}

	ips := make(map[string]bool)
	for range 2 {
		ip, err := r.AllocateNext()
		if err != nil {
			t.Fatalf("AllocateNext: %v", err)
		}
		ips[ip.String()] = true
	}

	if _, err := r.AllocateNext(); err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}

	// Verify allocated IPs are in the valid range
	if !ips["10.0.0.1"] || !ips["10.0.0.2"] {
		t.Fatalf("unexpected IPs allocated: %v", ips)
	}
}

func TestRange_Release(t *testing.T) {
	r, _ := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/24"))

	r.Allocate(net.ParseIP("10.0.0.1"))
	if r.Free() != 253 {
		t.Fatalf("expected 253 free, got %d", r.Free())
	}

	r.Release(net.ParseIP("10.0.0.1"))
	if r.Free() != 254 {
		t.Fatalf("expected 254 free after release, got %d", r.Free())
	}

	// Releasing out-of-range is a no-op
	r.Release(net.ParseIP("192.168.0.1"))
	if r.Free() != 254 {
		t.Fatalf("expected 254 free after no-op release, got %d", r.Free())
	}
}

func TestRange_ForEach(t *testing.T) {
	r, _ := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/24"))

	r.Allocate(net.ParseIP("10.0.0.1"))
	r.Allocate(net.ParseIP("10.0.0.100"))

	var collected []string
	r.ForEach(func(ip net.IP) {
		collected = append(collected, ip.String())
	})

	if len(collected) != 2 {
		t.Fatalf("expected 2 IPs, got %d", len(collected))
	}
	expected := map[string]bool{"10.0.0.1": true, "10.0.0.100": true}
	for _, ip := range collected {
		if !expected[ip] {
			t.Fatalf("unexpected IP %s in ForEach", ip)
		}
	}
}

func TestRange_Slash32(t *testing.T) {
	r, err := NewCIDRRange(mustParseCIDR(t, "10.0.0.1/32"))
	if err != nil {
		t.Fatalf("NewCIDRRange: %v", err)
	}

	// /32 = 1 address, usable (no exclusion)
	if r.Free() != 1 {
		t.Fatalf("expected 1 free for /32, got %d", r.Free())
	}

	if err := r.Allocate(net.ParseIP("10.0.0.1")); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if r.Free() != 0 {
		t.Fatalf("expected 0 free, got %d", r.Free())
	}
}

func TestRange_IPv6(t *testing.T) {
	r, err := NewCIDRRange(mustParseCIDR(t, "fd00::/120"))
	if err != nil {
		t.Fatalf("NewCIDRRange: %v", err)
	}

	// /120 = 256 addresses - 2 = 254 usable
	if r.Free() != 254 {
		t.Fatalf("expected 254 free for IPv6 /120, got %d", r.Free())
	}

	ip, err := r.AllocateNext()
	if err != nil {
		t.Fatalf("AllocateNext: %v", err)
	}
	if !r.Has(ip) {
		t.Fatalf("expected Has(%s) to be true", ip)
	}
}

func TestRange_FullAllocateRelease(t *testing.T) {
	r, _ := NewCIDRRange(mustParseCIDR(t, "10.0.0.0/24"))

	// Allocate all 254 IPs
	var allocated []net.IP
	for range 254 {
		ip, err := r.AllocateNext()
		if err != nil {
			t.Fatalf("AllocateNext: %v", err)
		}
		allocated = append(allocated, ip)
	}

	if r.Free() != 0 {
		t.Fatalf("expected 0 free, got %d", r.Free())
	}

	// Release all
	for _, ip := range allocated {
		r.Release(ip)
	}
	if r.Free() != 254 {
		t.Fatalf("expected 254 free, got %d", r.Free())
	}
}
