package probe

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// Phase identifies which stage of the connection failed.
type Phase string

const (
	PhaseDNS  Phase = "dns"
	PhaseTCP  Phase = "tcp"
	PhaseTLS  Phase = "tls"
	PhaseHTTP Phase = "http"
)

// Result is the unified return type for all probes.
type Result struct {
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	Phase      Phase         `json:"phase,omitempty"`
	Message    string        `json:"message,omitempty"`
	StatusCode int           `json:"statusCode,omitempty"`
}

// Options configures probe behavior.
type Options struct {
	Timeout time.Duration
}

// timeout returns the effective timeout from Options, defaulting to 5s.
func (o *Options) timeout() time.Duration {
	if o != nil && o.Timeout > 0 {
		return o.Timeout
	}
	return defaultTimeout
}

// classifyError determines which connection phase produced the error.
func classifyError(err error) Phase {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return PhaseDNS
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return PhaseTCP
	}

	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return PhaseTLS
	}
	if strings.Contains(err.Error(), "tls:") {
		return PhaseTLS
	}

	return PhaseTCP
}
