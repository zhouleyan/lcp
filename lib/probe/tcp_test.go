package probe

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCP_Success(t *testing.T) {
	// Start a local listener to accept connections.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	result := TCP(context.Background(), ln.Addr().String(), nil)
	if !result.Success {
		t.Errorf("TCP(%s) failed: %s", ln.Addr(), result.Message)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
	if result.Phase != "" {
		t.Errorf("expected empty phase on success, got %q", result.Phase)
	}
}

func TestTCP_ConnectionRefused(t *testing.T) {
	// Listen then immediately close to get a port that will refuse.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	result := TCP(context.Background(), addr, nil)
	if result.Success {
		t.Error("expected failure for closed port")
	}
	if result.Phase != PhaseTCP {
		t.Errorf("expected phase %q, got %q", PhaseTCP, result.Phase)
	}
}

func TestTCP_DNSFailure(t *testing.T) {
	host := "this.host.does.not.exist.invalid"
	// Skip if the local DNS resolver hijacks NXDOMAIN responses.
	if _, err := net.LookupHost(host); err == nil {
		t.Skipf("skipping: DNS resolver resolves %q (DNS hijacking detected)", host)
	}

	result := TCP(context.Background(), host+":9999", nil)
	if result.Success {
		t.Error("expected failure for unresolvable host")
	}
	if result.Phase != PhaseDNS {
		t.Errorf("expected phase %q, got %q", PhaseDNS, result.Phase)
	}
}

func TestTCP_Timeout(t *testing.T) {
	addr := "192.0.2.1:9999"
	// Skip if the TEST-NET-1 address is actually routable in this network.
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Skipf("skipping: %s is reachable (non-standard routing)", addr)
	}

	result := TCP(context.Background(), addr, &Options{Timeout: 200 * time.Millisecond})
	if result.Success {
		t.Error("expected failure for unreachable host")
	}
	if result.Duration < 150*time.Millisecond {
		t.Errorf("expected duration near timeout, got %v", result.Duration)
	}
}

func TestTCP_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Use any address; the canceled context should fail before connecting.
	result := TCP(ctx, "127.0.0.1:1", nil)
	if result.Success {
		t.Error("expected failure for canceled context")
	}
}
