package ipam

import (
	"fmt"
	"math/big"
	"net"
)

// bigForIP creates a big.Int based on the provided net.IP.
func bigForIP(ip net.IP) *big.Int {
	// Convert to 16-byte representation so we can handle v4 and v6 the same way.
	return big.NewInt(0).SetBytes(ip.To16())
}

// addIPOffset adds the provided integer offset to a base big.Int representing a net.IP.
func addIPOffset(base *big.Int, offset int) net.IP {
	r := big.NewInt(0).Add(base, big.NewInt(int64(offset))).Bytes()
	r = append(make([]byte, 16), r...)
	return net.IP(r[len(r)-16:])
}

// calculateIPOffset calculates the integer offset of ip from base such that
// base + offset = ip. It requires ip >= base.
func calculateIPOffset(base *big.Int, ip net.IP) int {
	return int(big.NewInt(0).Sub(bigForIP(ip), base).Int64())
}

// RangeSize returns the number of addresses in a subnet.
// For IPv6 subnets larger than /112, the size is capped at 65536.
func RangeSize(subnet *net.IPNet) int64 {
	ones, bits := subnet.Mask.Size()
	if bits == 32 && (bits-ones) >= 31 || bits == 128 && (bits-ones) >= 127 {
		return 0
	}
	// For IPv6, cap the max size at 65536 to keep bitmap size reasonable.
	if bits == 128 && (bits-ones) >= 16 {
		return int64(1) << 16
	}
	return int64(1) << uint(bits-ones)
}

// GetIndexedIP returns a net.IP that is subnet.IP + index in the contiguous IP space.
func GetIndexedIP(subnet *net.IPNet, index int) (net.IP, error) {
	ip := addIPOffset(bigForIP(subnet.IP), index)
	if !subnet.Contains(ip) {
		return nil, fmt.Errorf("can't generate IP with index %d from subnet. subnet too small. subnet: %q", index, subnet)
	}
	return ip, nil
}
