package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTP_Success200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := HTTP(context.Background(), srv.URL, nil)
	if !result.Success {
		t.Errorf("HTTP(%s) failed: %s", srv.URL, result.Message)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
}

func TestHTTP_Success4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	result := HTTP(context.Background(), srv.URL, nil)
	if !result.Success {
		t.Errorf("HTTP(%s) should succeed for 401 (service is alive): %s", srv.URL, result.Message)
	}
	if result.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", result.StatusCode)
	}
}

func TestHTTP_Fail5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	result := HTTP(context.Background(), srv.URL, nil)
	if result.Success {
		t.Error("expected failure for 503")
	}
	if result.Phase != PhaseHTTP {
		t.Errorf("expected phase %q, got %q", PhaseHTTP, result.Phase)
	}
	if result.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", result.StatusCode)
	}
}

func TestHTTP_TLSSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := HTTP(context.Background(), srv.URL, nil)
	if !result.Success {
		t.Errorf("HTTPS(%s) failed: %s", srv.URL, result.Message)
	}
}

func TestHTTP_ConnectionRefused(t *testing.T) {
	result := HTTP(context.Background(), "http://127.0.0.1:1", nil)
	if result.Success {
		t.Error("expected failure for unreachable server")
	}
	if result.Phase != PhaseTCP {
		t.Errorf("expected phase %q, got %q", PhaseTCP, result.Phase)
	}
}

func TestHTTP_DNSFailure(t *testing.T) {
	host := "this.host.does.not.exist.invalid"
	// Skip if the local DNS resolver hijacks NXDOMAIN responses.
	if _, err := net.LookupHost(host); err == nil {
		t.Skipf("skipping: DNS resolver resolves %q (DNS hijacking detected)", host)
	}

	result := HTTP(context.Background(), "http://"+host+"/health", nil)
	if result.Success {
		t.Error("expected failure for unresolvable host")
	}
	if result.Phase != PhaseDNS {
		t.Errorf("expected phase %q, got %q", PhaseDNS, result.Phase)
	}
}

func TestHTTP_Timeout(t *testing.T) {
	addr := "192.0.2.1:9999"
	// Skip if the TEST-NET-1 address is actually routable in this network.
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Skipf("skipping: %s is reachable (non-standard routing)", addr)
	}

	result := HTTP(context.Background(), "http://"+addr+"/health", &Options{Timeout: 200 * time.Millisecond})
	if result.Success {
		t.Error("expected failure for unreachable host")
	}
	if result.Duration < 150*time.Millisecond {
		t.Errorf("expected duration near timeout, got %v", result.Duration)
	}
}

func TestHTTP_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use 127.0.0.1:1 instead of 192.0.2.1 to avoid routing issues.
	result := HTTP(ctx, "http://127.0.0.1:1/health", nil)
	if result.Success {
		t.Error("expected failure for canceled context")
	}
}

func TestHTTP_InvalidURL(t *testing.T) {
	result := HTTP(context.Background(), "://bad-url", nil)
	if result.Success {
		t.Error("expected failure for invalid URL")
	}
}
