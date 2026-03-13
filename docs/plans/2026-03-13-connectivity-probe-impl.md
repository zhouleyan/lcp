# Connectivity Probe Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `lib/probe/` package providing TCP and HTTP connectivity probing with phase-aware error reporting.

**Architecture:** Flat package `lib/probe/` with three files: shared types (`probe.go`), TCP prober (`tcp.go`), HTTP prober (`http.go`). Pure standard library, no external dependencies. TDD — tests first, then implementation.

**Tech Stack:** Go standard library (`net`, `net/http`, `crypto/tls`, `errors`, `testing`, `net/http/httptest`)

**Design doc:** `docs/plans/2026-03-13-connectivity-probe-design.md`

---

### Task 1: Core types — probe.go

**Files:**
- Create: `lib/probe/probe.go`
- Test: `lib/probe/probe_test.go`

**Step 1: Write the failing test for classifyError**

Create `lib/probe/probe_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/probe/ -run TestClassifyError -v`
Expected: FAIL — `classifyError` not defined

**Step 3: Write probe.go with types and classifyError**

Create `lib/probe/probe.go`:

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/probe/ -run TestClassifyError -v`
Expected: PASS (all 4 tests)

**Step 5: Commit**

```bash
git add lib/probe/probe.go lib/probe/probe_test.go
git commit -m "feat(probe): add core types and classifyError"
```

---

### Task 2: TCP probe — tcp.go

**Files:**
- Create: `lib/probe/tcp.go`
- Test: `lib/probe/tcp_test.go`

**Step 1: Write the failing tests**

Create `lib/probe/tcp_test.go`:

```go
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
	result := TCP(context.Background(), "this.host.does.not.exist.invalid:9999", nil)
	if result.Success {
		t.Error("expected failure for unresolvable host")
	}
	if result.Phase != PhaseDNS {
		t.Errorf("expected phase %q, got %q", PhaseDNS, result.Phase)
	}
}

func TestTCP_Timeout(t *testing.T) {
	result := TCP(context.Background(), "192.0.2.1:9999", &Options{Timeout: 200 * time.Millisecond})
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

	result := TCP(ctx, "192.0.2.1:9999", nil)
	if result.Success {
		t.Error("expected failure for canceled context")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/probe/ -run TestTCP -v`
Expected: FAIL — `TCP` not defined

**Step 3: Write tcp.go**

Create `lib/probe/tcp.go`:

```go
package probe

import (
	"context"
	"net"
	"time"
)

// TCP probes whether the given addr (host:port) is reachable via TCP.
func TCP(ctx context.Context, addr string, opts *Options) *Result {
	timeout := opts.timeout()

	start := time.Now()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	duration := time.Since(start)

	if err != nil {
		return &Result{
			Success:  false,
			Duration: duration,
			Phase:    classifyError(err),
			Message:  err.Error(),
		}
	}
	conn.Close()

	return &Result{
		Success:  true,
		Duration: duration,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/probe/ -run TestTCP -v`
Expected: PASS (all 5 tests). The timeout test takes ~200ms.

**Step 5: Commit**

```bash
git add lib/probe/tcp.go lib/probe/tcp_test.go
git commit -m "feat(probe): add TCP connectivity probe"
```

---

### Task 3: HTTP probe — http.go

**Files:**
- Create: `lib/probe/http.go`
- Test: `lib/probe/http_test.go`

**Step 1: Write the failing tests**

Create `lib/probe/http_test.go`:

```go
package probe

import (
	"context"
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
	result := HTTP(context.Background(), "http://this.host.does.not.exist.invalid/health", nil)
	if result.Success {
		t.Error("expected failure for unresolvable host")
	}
	if result.Phase != PhaseDNS {
		t.Errorf("expected phase %q, got %q", PhaseDNS, result.Phase)
	}
}

func TestHTTP_Timeout(t *testing.T) {
	result := HTTP(context.Background(), "http://192.0.2.1:9999/health", &Options{Timeout: 200 * time.Millisecond})
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

	result := HTTP(ctx, "http://192.0.2.1:9999/health", nil)
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/probe/ -run TestHTTP -v`
Expected: FAIL — `HTTP` not defined

**Step 3: Write http.go**

Create `lib/probe/http.go`:

```go
package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// HTTP probes whether the given URL is reachable and returns a non-5xx status.
func HTTP(ctx context.Context, url string, opts *Options) *Result {
	timeout := opts.timeout()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout: timeout,
			}).DialContext,
		},
		// Do not follow redirects — just record the first response.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &Result{
			Success: false,
			Phase:   PhaseTCP,
			Message: fmt.Sprintf("invalid request: %v", err),
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &Result{
			Success:  false,
			Duration: duration,
			Phase:    classifyError(err),
			Message:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &Result{
			Success:    false,
			Duration:   duration,
			Phase:      PhaseHTTP,
			Message:    fmt.Sprintf("server error: %s", resp.Status),
			StatusCode: resp.StatusCode,
		}
	}

	return &Result{
		Success:    true,
		Duration:   duration,
		StatusCode: resp.StatusCode,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/probe/ -run TestHTTP -v`
Expected: PASS (all 9 tests). The timeout test takes ~200ms.

**Step 5: Run all tests together**

Run: `go test ./lib/probe/ -v`
Expected: PASS — all tests across probe_test.go, tcp_test.go, http_test.go

**Step 6: Commit**

```bash
git add lib/probe/http.go lib/probe/http_test.go
git commit -m "feat(probe): add HTTP connectivity probe"
```

---

### Task 4: Final verification

**Step 1: Run full test suite**

Run: `go test ./lib/probe/ -v -count=1`
Expected: PASS — all tests green

**Step 2: Run vet and lint**

Run: `go vet ./lib/probe/`
Expected: No issues

Run: `make lint` (if golangci-lint is available)
Expected: No issues in `lib/probe/`

**Step 3: Run project-wide tests to ensure no breakage**

Run: `make test`
Expected: PASS
