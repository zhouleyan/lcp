package probe

import (
	"crypto/tls"
	"net"
	"testing"
)

func TestClassifyError_DNSError(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "bad.invalid"}
	if got := classifyError(err); got != PhaseDNS {
		t.Errorf("classifyError(DNSError) = %q, want %q", got, PhaseDNS)
	}
}

func TestClassifyError_OpError(t *testing.T) {
	err := &net.OpError{Op: "dial", Err: &net.DNSError{Err: "connection refused"}}
	// DNSError is nested inside OpError — DNSError should win (checked first)
	if got := classifyError(err); got != PhaseDNS {
		t.Errorf("classifyError(OpError wrapping DNSError) = %q, want %q", got, PhaseDNS)
	}

	// Pure OpError without DNS
	err2 := &net.OpError{Op: "dial", Err: &net.AddrError{Err: "connection refused"}}
	if got := classifyError(err2); got != PhaseTCP {
		t.Errorf("classifyError(OpError) = %q, want %q", got, PhaseTCP)
	}
}

func TestClassifyError_TLSError(t *testing.T) {
	err := &tls.CertificateVerificationError{}
	if got := classifyError(err); got != PhaseTLS {
		t.Errorf("classifyError(CertificateVerificationError) = %q, want %q", got, PhaseTLS)
	}
}

func TestClassifyError_FallbackTCP(t *testing.T) {
	err := &net.AddrError{Err: "some error"}
	if got := classifyError(err); got != PhaseTCP {
		t.Errorf("classifyError(unknown) = %q, want %q", got, PhaseTCP)
	}
}
