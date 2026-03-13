# SSH ProxyJump Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add bastion/jump host support to `lib/clients/sshclient` via `Config.ProxyJump` field, with automatic lifecycle management and multi-level chaining.

**Architecture:** Add `ProxyJump *Config` to Config, add `proxyClient *Client` and `proxyConn net.Conn` to Client. `Connect()` recursively connects through the chain, `Close()` cascades in reverse. Zero changes to command.go or sftp.go.

**Tech Stack:** `golang.org/x/crypto/ssh` (existing dependency)

**Design doc:** `docs/plans/2026-03-13-sshclient-proxyjump-design.md`

---

### Task 1: Add ProxyJump field to Config and proxy fields to Client

**Files:**
- Modify: `lib/clients/sshclient/client.go:31-48` (Config struct)
- Modify: `lib/clients/sshclient/client.go:63-68` (Client struct)
- Test: `lib/clients/sshclient/client_test.go`

**Step 1: Write the failing test**

Add to `lib/clients/sshclient/client_test.go`:

```go
func TestNewWithProxyJump(t *testing.T) {
	c := New(Config{
		Host:     "target.example.com",
		Password: "target-pass",
		ProxyJump: &Config{
			Host:     "bastion.example.com",
			Password: "bastion-pass",
		},
	})
	if c.config.ProxyJump == nil {
		t.Fatal("expected ProxyJump config to be set")
	}
	if c.config.ProxyJump.Host != "bastion.example.com" {
		t.Errorf("expected bastion host, got %q", c.config.ProxyJump.Host)
	}
	// ProxyJump config should also have defaults applied
	if c.config.ProxyJump.Port != 22 {
		t.Errorf("expected default port 22 on ProxyJump, got %d", c.config.ProxyJump.Port)
	}
	if c.config.ProxyJump.User != "root" {
		t.Errorf("expected default user on ProxyJump, got %q", c.config.ProxyJump.User)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/clients/sshclient/ -run TestNewWithProxyJump -v`
Expected: FAIL — `Config` has no field `ProxyJump`

**Step 3: Add ProxyJump to Config, proxy fields to Client, update New()**

In `lib/clients/sshclient/client.go`, add to Config struct (after Timeout field):

```go
	// ProxyJump is the bastion/jump host config. When set, the client
	// connects to Host through the proxy via SSH tunnel. Supports
	// multi-level chaining (ProxyJump can itself have a ProxyJump).
	// Nil means direct connection.
	ProxyJump *Config
```

Add to Client struct (after `client` field):

```go
	proxyClient *Client  // bastion client (auto-managed lifecycle)
	proxyConn   net.Conn // tunnel connection through bastion
```

Add `"net"` to imports.

Update `New()` to apply defaults on ProxyJump:

```go
func New(config Config) *Client {
	config.applyDefaults()
	if config.ProxyJump != nil {
		config.ProxyJump.applyDefaults()
	}
	return &Client{
		config: config,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/clients/sshclient/ -run TestNewWithProxyJump -v`
Expected: PASS

**Step 5: Run all existing tests to verify no breakage**

Run: `go test ./lib/clients/sshclient/ -v`
Expected: All existing tests PASS

**Step 6: Commit**

```bash
git add lib/clients/sshclient/client.go lib/clients/sshclient/client_test.go
git commit -m "feat(sshclient): add ProxyJump field to Config"
```

---

### Task 2: Implement ProxyJump in Connect()

**Files:**
- Modify: `lib/clients/sshclient/client.go:87-126` (Connect method)
- Test: `lib/clients/sshclient/client_test.go`

**Step 1: Write the failing test**

This test needs a mock SSH server. Add a test helper and the test to `lib/clients/sshclient/client_test.go`:

```go
import (
	// add these to existing imports
	"context"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer starts a minimal SSH server that accepts connections
// with password auth. It returns the listener address and a cleanup function.
// If allowTunnel is true, the server handles "direct-tcpip" channel requests
// by dialing the requested address (acting as a bastion).
func startTestSSHServer(t *testing.T, password string, allowTunnel bool) (addr string, cleanup func()) {
	t.Helper()

	// Generate a host key for the server.
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("failed to create host signer: %v", err)
	}

	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go handleTestSSHConn(conn, serverConfig, allowTunnel)
		}
	}()

	return ln.Addr().String(), func() { ln.Close() }
}

func handleTestSSHConn(conn net.Conn, config *ssh.ServerConfig, allowTunnel bool) {
	srvConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		conn.Close()
		return
	}
	defer srvConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		switch newChan.ChannelType() {
		case "session":
			ch, _, err := newChan.Accept()
			if err != nil {
				continue
			}
			// Just close the session channel — enough for Connect() to succeed.
			ch.Close()

		case "direct-tcpip":
			if !allowTunnel {
				newChan.Reject(ssh.Prohibited, "tunneling not allowed")
				continue
			}

			// Parse the target address from the extra data.
			var payload struct {
				DestHost string
				DestPort uint32
				SrcHost  string
				SrcPort  uint32
			}
			if err := ssh.Unmarshal(newChan.ExtraData(), &payload); err != nil {
				newChan.Reject(ssh.ConnectionFailed, "bad payload")
				continue
			}

			targetAddr := fmt.Sprintf("%s:%d", payload.DestHost, payload.DestPort)
			targetConn, err := net.Dial("tcp", targetAddr)
			if err != nil {
				newChan.Reject(ssh.ConnectionFailed, err.Error())
				continue
			}

			ch, _, err := newChan.Accept()
			if err != nil {
				targetConn.Close()
				continue
			}

			// Bidirectional copy.
			go func() {
				defer ch.Close()
				defer targetConn.Close()
				go func() { _, _ = io.Copy(ch, targetConn) }()
				_, _ = io.Copy(targetConn, ch)
			}()

		default:
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
		}
	}
}

func TestConnect_ProxyJump(t *testing.T) {
	// Start target SSH server.
	targetAddr, targetCleanup := startTestSSHServer(t, "target-pass", false)
	defer targetCleanup()

	// Start bastion SSH server with tunneling enabled.
	bastionAddr, bastionCleanup := startTestSSHServer(t, "bastion-pass", true)
	defer bastionCleanup()

	// Parse bastion host:port.
	bastionHost, bastionPortStr, _ := net.SplitHostPort(bastionAddr)
	bastionPort := 0
	fmt.Sscanf(bastionPortStr, "%d", &bastionPort)

	// Parse target host:port.
	targetHost, targetPortStr, _ := net.SplitHostPort(targetAddr)
	targetPort := 0
	fmt.Sscanf(targetPortStr, "%d", &targetPort)

	c := New(Config{
		Host:     targetHost,
		Port:     targetPort,
		Password: "target-pass",
		ProxyJump: &Config{
			Host:     bastionHost,
			Port:     bastionPort,
			Password: "bastion-pass",
		},
	})

	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect via ProxyJump failed: %v", err)
	}
	defer c.Close()

	// Verify internal state.
	if c.SSHClient() == nil {
		t.Error("expected target SSH client to be set")
	}
	if c.proxyClient == nil {
		t.Error("expected proxyClient to be set")
	}
	if c.proxyConn == nil {
		t.Error("expected proxyConn to be set")
	}
}
```

Add `"io"` to imports.

**Step 2: Run test to verify it fails**

Run: `go test ./lib/clients/sshclient/ -run TestConnect_ProxyJump -v`
Expected: FAIL — `Connect` does not handle `ProxyJump`

**Step 3: Implement ProxyJump logic in Connect()**

Replace the `Connect()` method in `lib/clients/sshclient/client.go` with:

```go
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Host == "" {
		return fmt.Errorf("sshclient: host is not set")
	}

	auth, err := c.buildAuthMethods()
	if err != nil {
		return err
	}
	if len(auth) == 0 {
		return fmt.Errorf("sshclient: no authentication method available: provide password, private_key_content, or private_key")
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.config.Timeout,
	}

	// Use context deadline if available to set a shorter timeout.
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < sshConfig.Timeout {
			sshConfig.Timeout = remaining
		}
	}

	if c.config.ProxyJump != nil {
		// Connect through bastion host.
		proxy := New(*c.config.ProxyJump)
		if err := proxy.Connect(ctx); err != nil {
			return fmt.Errorf("sshclient: failed to connect proxy %s:%d: %w",
				c.config.ProxyJump.Host, c.config.ProxyJump.Port, err)
		}

		tunnelConn, err := proxy.client.Dial("tcp", addr)
		if err != nil {
			proxy.Close()
			return fmt.Errorf("sshclient: failed to tunnel to %s: %w", addr, err)
		}

		ncc, chans, reqs, err := ssh.NewClientConn(tunnelConn, addr, sshConfig)
		if err != nil {
			tunnelConn.Close()
			proxy.Close()
			return fmt.Errorf("sshclient: failed to dial %s via proxy: %w", addr, err)
		}

		c.client = ssh.NewClient(ncc, chans, reqs)
		c.proxyClient = proxy
		c.proxyConn = tunnelConn
	} else {
		// Direct connection.
		sshClient, err := ssh.Dial("tcp", addr, sshConfig)
		if err != nil {
			return fmt.Errorf("sshclient: failed to dial %s: %w", addr, err)
		}
		c.client = sshClient
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/clients/sshclient/ -run TestConnect_ProxyJump -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./lib/clients/sshclient/ -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add lib/clients/sshclient/client.go lib/clients/sshclient/client_test.go
git commit -m "feat(sshclient): implement ProxyJump tunnel in Connect()"
```

---

### Task 3: Implement cascade Close()

**Files:**
- Modify: `lib/clients/sshclient/client.go:130-140` (Close method)
- Test: `lib/clients/sshclient/client_test.go`

**Step 1: Write the failing test**

Add to `lib/clients/sshclient/client_test.go`:

```go
func TestClose_ProxyJumpCascade(t *testing.T) {
	// Start target and bastion servers.
	targetAddr, targetCleanup := startTestSSHServer(t, "target-pass", false)
	defer targetCleanup()

	bastionAddr, bastionCleanup := startTestSSHServer(t, "bastion-pass", true)
	defer bastionCleanup()

	bastionHost, bastionPortStr, _ := net.SplitHostPort(bastionAddr)
	bastionPort := 0
	fmt.Sscanf(bastionPortStr, "%d", &bastionPort)

	targetHost, targetPortStr, _ := net.SplitHostPort(targetAddr)
	targetPort := 0
	fmt.Sscanf(targetPortStr, "%d", &targetPort)

	c := New(Config{
		Host:     targetHost,
		Port:     targetPort,
		Password: "target-pass",
		ProxyJump: &Config{
			Host:     bastionHost,
			Port:     bastionPort,
			Password: "bastion-pass",
		},
	})

	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Close should clean up everything.
	if err := c.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify all resources are nil after close.
	if c.SSHClient() != nil {
		t.Error("expected client to be nil after Close")
	}
	if c.proxyClient != nil {
		t.Error("expected proxyClient to be nil after Close")
	}
	if c.proxyConn != nil {
		t.Error("expected proxyConn to be nil after Close")
	}

	// Close again should be safe (idempotent).
	if err := c.Close(); err != nil {
		t.Fatalf("second Close should not error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/clients/sshclient/ -run TestClose_ProxyJumpCascade -v`
Expected: FAIL — `proxyClient` and `proxyConn` are not cleaned up

**Step 3: Update Close() for cascade cleanup**

Replace the `Close()` method in `lib/clients/sshclient/client.go`:

```go
// Close closes the underlying SSH connection and any proxy resources.
// It is safe to call multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close target SSH connection.
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	// Close tunnel connection.
	if c.proxyConn != nil {
		c.proxyConn.Close()
		c.proxyConn = nil
	}

	// Cascade close to bastion client.
	if c.proxyClient != nil {
		c.proxyClient.Close()
		c.proxyClient = nil
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./lib/clients/sshclient/ -run TestClose_ProxyJumpCascade -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./lib/clients/sshclient/ -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add lib/clients/sshclient/client.go lib/clients/sshclient/client_test.go
git commit -m "feat(sshclient): implement cascade Close() for ProxyJump"
```

---

### Task 4: Add error case tests and update README

**Files:**
- Modify: `lib/clients/sshclient/client_test.go`
- Modify: `lib/clients/sshclient/README.md`

**Step 1: Write error case tests**

Add to `lib/clients/sshclient/client_test.go`:

```go
func TestConnect_ProxyJumpBastionFails(t *testing.T) {
	// Bastion is unreachable — Connect should fail with proxy error.
	c := New(Config{
		Host:     "127.0.0.1",
		Port:     1,
		Password: "target-pass",
		ProxyJump: &Config{
			Host:     "127.0.0.1",
			Port:     1, // nothing listening
			Password: "bastion-pass",
		},
	})

	err := c.Connect(context.Background())
	if err == nil {
		c.Close()
		t.Fatal("expected error when bastion is unreachable")
	}
	// Should mention proxy in error.
	if !contains(err.Error(), "proxy") {
		t.Errorf("expected proxy-related error, got: %v", err)
	}
}

func TestConnect_ProxyJumpBadTargetPassword(t *testing.T) {
	// Bastion works, but target password is wrong.
	targetAddr, targetCleanup := startTestSSHServer(t, "correct-pass", false)
	defer targetCleanup()

	bastionAddr, bastionCleanup := startTestSSHServer(t, "bastion-pass", true)
	defer bastionCleanup()

	bastionHost, bastionPortStr, _ := net.SplitHostPort(bastionAddr)
	bastionPort := 0
	fmt.Sscanf(bastionPortStr, "%d", &bastionPort)

	targetHost, targetPortStr, _ := net.SplitHostPort(targetAddr)
	targetPort := 0
	fmt.Sscanf(targetPortStr, "%d", &targetPort)

	c := New(Config{
		Host:     targetHost,
		Port:     targetPort,
		Password: "wrong-pass",
		ProxyJump: &Config{
			Host:     bastionHost,
			Port:     bastionPort,
			Password: "bastion-pass",
		},
	})

	err := c.Connect(context.Background())
	if err == nil {
		c.Close()
		t.Fatal("expected error when target auth fails")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

Note: Replace the `contains`/`containsSubstr` helpers with `strings.Contains` by adding `"strings"` to imports:

```go
import "strings"

// Then use: strings.Contains(err.Error(), "proxy")
```

**Step 2: Run tests to verify they pass**

Run: `go test ./lib/clients/sshclient/ -run "TestConnect_ProxyJump" -v`
Expected: All ProxyJump tests PASS

**Step 3: Update README with ProxyJump usage**

Add the following section to `lib/clients/sshclient/README.md` after the "认证方式" section:

```markdown
## 堡垒机跳转（ProxyJump）

通过堡垒机/跳板机访问受保护网络中的目标主机，等价于 `ssh -J bastion target`：

\```go
client := sshclient.New(sshclient.Config{
    Host:     "10.0.0.5",
    Password: "target-pass",
    ProxyJump: &sshclient.Config{
        Host:     "bastion.example.com",
        Password: "bastion-pass",
    },
})

if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer client.Close() // 自动级联关闭：目标连接 → 隧道 → 堡垒机
\```

支持多级跳转（递归嵌套）：

\```go
client := sshclient.New(sshclient.Config{
    Host:     "10.0.0.5",
    Password: "target-pass",
    ProxyJump: &sshclient.Config{
        Host:     "bastion2",
        Password: "pass2",
        ProxyJump: &sshclient.Config{
            Host:     "bastion1",
            Password: "pass1",
        },
    },
})
\```

连接建立后，`Exec`、`ExecWithSudo`、`PutFile`、`FetchFile` 等操作与直连完全一致。
```

**Step 4: Run full test suite**

Run: `go test ./lib/clients/sshclient/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add lib/clients/sshclient/client_test.go lib/clients/sshclient/README.md
git commit -m "test(sshclient): add ProxyJump error case tests and update README"
```

---

### Task 5: Final verification

**Step 1: Run full sshclient test suite**

Run: `go test ./lib/clients/sshclient/ -v -count=1`
Expected: All tests PASS

**Step 2: Run vet**

Run: `go vet ./lib/clients/sshclient/`
Expected: No issues

**Step 3: Run project-wide tests**

Run: `make test`
Expected: PASS
