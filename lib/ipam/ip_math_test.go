package ipam

import (
	"math/big"
	"net"
	"testing"
)

func TestBigForIP(t *testing.T) {
	// bigForIP uses 16-byte (IPv6-mapped) representation, so IPv4 values
	// include the ::ffff: prefix. We test round-trip via addIPOffset instead.
	base := bigForIP(net.ParseIP("10.0.0.0"))
	ip := addIPOffset(base, 1)
	if ip.String() != "10.0.0.1" {
		t.Errorf("round-trip bigForIP+addIPOffset: got %s, want 10.0.0.1", ip.String())
	}

	// Verify offset calculation is consistent
	offset := calculateIPOffset(base, net.ParseIP("10.0.0.5"))
	if offset != 5 {
		t.Errorf("calculateIPOffset(10.0.0.0, 10.0.0.5) = %d, want 5", offset)
	}
}

func TestAddIPOffset(t *testing.T) {
	base := big.NewInt(0x0a000001) // 10.0.0.1
	// Need to construct the same way bigForIP does (16-byte)
	base = bigForIP(net.ParseIP("10.0.0.1"))

	ip := addIPOffset(base, 5)
	expected := "10.0.0.6"
	if ip.String() != expected {
		t.Errorf("addIPOffset(10.0.0.1, 5) = %s, want %s", ip.String(), expected)
	}
}

func TestCalculateIPOffset(t *testing.T) {
	base := bigForIP(net.ParseIP("10.0.0.1"))
	ip := net.ParseIP("10.0.0.10")
	offset := calculateIPOffset(base, ip)
	if offset != 9 {
		t.Errorf("calculateIPOffset(10.0.0.1, 10.0.0.10) = %d, want 9", offset)
	}
}

func TestRangeSize(t *testing.T) {
	tests := []struct {
		cidr     string
		expected int64
	}{
		{"10.0.0.0/24", 256},
		{"10.0.0.0/20", 4096},
		{"10.0.0.0/32", 1},
		{"10.0.0.0/31", 2},
		{"fd00::/120", 256},
		{"fd00::/112", 65536},
		{"fd00::/64", 65536}, // capped at 65536 for large IPv6
	}
	for _, tt := range tests {
		_, subnet, err := net.ParseCIDR(tt.cidr)
		if err != nil {
			t.Fatalf("ParseCIDR(%s): %v", tt.cidr, err)
		}
		got := RangeSize(subnet)
		if got != tt.expected {
			t.Errorf("RangeSize(%s) = %d, want %d", tt.cidr, got, tt.expected)
		}
	}
}

func TestGetIndexedIP(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("10.0.0.0/24")

	ip, err := GetIndexedIP(subnet, 1)
	if err != nil {
		t.Fatalf("GetIndexedIP: %v", err)
	}
	if ip.String() != "10.0.0.1" {
		t.Errorf("GetIndexedIP(10.0.0.0/24, 1) = %s, want 10.0.0.1", ip.String())
	}

	ip, err = GetIndexedIP(subnet, 255)
	if err != nil {
		t.Fatalf("GetIndexedIP: %v", err)
	}
	if ip.String() != "10.0.0.255" {
		t.Errorf("GetIndexedIP(10.0.0.0/24, 255) = %s, want 10.0.0.255", ip.String())
	}

	// Out of range
	_, err = GetIndexedIP(subnet, 256)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}
