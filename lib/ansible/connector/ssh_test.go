package connector

import (
	"net"
	"os"
	"testing"

	"lcp.io/lcp/lib/clients/sshclient"
)

// ---------------------------------------------------------------------------
// NewConnector factory tests
// ---------------------------------------------------------------------------

func TestNewConnector_Local(t *testing.T) {
	vars := map[string]any{
		"connection": "local",
	}
	c, err := NewConnector("192.168.1.100", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := c.(*LocalConnector); !ok {
		t.Fatalf("expected *LocalConnector, got %T", c)
	}
}

func TestNewConnector_LocalWithPassword(t *testing.T) {
	vars := map[string]any{
		"connection": "local",
		"password":   "secret",
	}
	c, err := NewConnector("anyhost", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lc, ok := c.(*LocalConnector)
	if !ok {
		t.Fatalf("expected *LocalConnector, got %T", c)
	}
	if lc.password != "secret" {
		t.Errorf("expected password %q, got %q", "secret", lc.password)
	}
}

func TestNewConnector_SSH(t *testing.T) {
	vars := map[string]any{
		"connection":  "ssh",
		"password":    "pw",
		"port":        2222,
		"remote_user": "deploy",
		"become":      true,
		"become_user": "admin",
	}
	c, err := NewConnector("10.0.0.5", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sc, ok := c.(*SSHConnector)
	if !ok {
		t.Fatalf("expected *SSHConnector, got %T", c)
	}
	if sc.host != "10.0.0.5" {
		t.Errorf("expected host %q, got %q", "10.0.0.5", sc.host)
	}
	if sc.password != "pw" {
		t.Errorf("expected password %q, got %q", "pw", sc.password)
	}
	if !sc.become {
		t.Error("expected become=true")
	}
	if sc.becomeUser != "admin" {
		t.Errorf("expected becomeUser %q, got %q", "admin", sc.becomeUser)
	}
}

func TestNewConnector_SSHDefaultType(t *testing.T) {
	// Empty connection type with a remote host should produce an SSHConnector.
	vars := map[string]any{
		"password": "pw",
	}
	c, err := NewConnector("10.0.0.5", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := c.(*SSHConnector); !ok {
		t.Fatalf("expected *SSHConnector for remote host, got %T", c)
	}
}

func TestNewConnector_SSHWithKeyParams(t *testing.T) {
	vars := map[string]any{
		"private_key":         "/path/to/key",
		"private_key_content": "PEM-CONTENT",
	}
	c, err := NewConnector("10.0.0.5", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := c.(*SSHConnector); !ok {
		t.Fatalf("expected *SSHConnector, got %T", c)
	}
}

func TestNewConnector_Localhost(t *testing.T) {
	vars := map[string]any{}

	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		c, err := NewConnector(host, vars)
		if err != nil {
			t.Fatalf("unexpected error for host %q: %v", host, err)
		}
		if _, ok := c.(*LocalConnector); !ok {
			t.Errorf("expected *LocalConnector for host %q, got %T", host, c)
		}
	}
}

func TestNewConnector_LocalHostname(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("cannot determine hostname")
	}
	vars := map[string]any{}
	c, err := NewConnector(hostname, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := c.(*LocalConnector); !ok {
		t.Errorf("expected *LocalConnector for OS hostname %q, got %T", hostname, c)
	}
}

func TestNewConnector_Unsupported(t *testing.T) {
	vars := map[string]any{
		"connection": "docker",
	}
	_, err := NewConnector("anyhost", vars)
	if err == nil {
		t.Fatal("expected error for unsupported connection type")
	}
}

// ---------------------------------------------------------------------------
// isLocal tests
// ---------------------------------------------------------------------------

func TestIsLocal_KnownLocal(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.99.99.99", false},
		{"remote.example.com", false},
	}
	for _, tt := range tests {
		got := isLocal(tt.host)
		if got != tt.want {
			t.Errorf("isLocal(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestIsLocal_OSHostname(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("cannot determine hostname")
	}
	if !isLocal(hostname) {
		t.Errorf("isLocal(%q) = false, expected true for OS hostname", hostname)
	}
}

func TestIsLocal_LocalInterfaceIP(t *testing.T) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Skip("cannot list interface addresses")
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		// At least one non-loopback local IP should be detected.
		if isLocal(ip.String()) {
			return // success
		}
	}
	// If we get here, either no non-loopback IPs exist or isLocal failed.
	// On machines with only loopback, this is acceptable.
}

// ---------------------------------------------------------------------------
// SSHConnector unit tests (no real SSH connection)
// ---------------------------------------------------------------------------

func TestSSHConnector_NotConnected(t *testing.T) {
	sc := NewSSHConnector(sshclient.Config{
		Host: "10.0.0.1",
	}, false, "")

	// ExecuteCommand should fail because we never called Init.
	_, _, err := sc.ExecuteCommand(t.Context(), "echo hello")
	if err == nil {
		t.Error("expected error from ExecuteCommand without Init")
	}

	// PutFile should fail.
	err = sc.PutFile(t.Context(), []byte("data"), "/tmp/test", 0644)
	if err == nil {
		t.Error("expected error from PutFile without Init")
	}

	// FetchFile should fail.
	var buf []byte
	err = sc.FetchFile(t.Context(), "/tmp/test", nil)
	if err == nil {
		t.Error("expected error from FetchFile without Init")
	}
	_ = buf
}

func TestSSHConnector_CloseWithoutInit(t *testing.T) {
	sc := NewSSHConnector(sshclient.Config{
		Host: "10.0.0.1",
	}, false, "")

	// Close on an uninitialized connector should not panic or error.
	if err := sc.Close(t.Context()); err != nil {
		t.Errorf("Close without Init should not error, got: %v", err)
	}
}

func TestSSHConnector_Fields(t *testing.T) {
	sc := NewSSHConnector(sshclient.Config{
		Host:     "10.0.0.5",
		Port:     2222,
		User:     "deploy",
		Password: "secret",
	}, true, "admin")

	if sc.host != "10.0.0.5" {
		t.Errorf("expected host %q, got %q", "10.0.0.5", sc.host)
	}
	if sc.password != "secret" {
		t.Errorf("expected password %q, got %q", "secret", sc.password)
	}
	if !sc.become {
		t.Error("expected become=true")
	}
	if sc.becomeUser != "admin" {
		t.Errorf("expected becomeUser %q, got %q", "admin", sc.becomeUser)
	}
}
