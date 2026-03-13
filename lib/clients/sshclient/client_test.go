package sshclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// generateTestKeyPEM generates an Ed25519 private key and returns the
// PEM-encoded bytes suitable for SSH authentication.
func generateTestKeyPEM(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	return pem.EncodeToMemory(block)
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{Host: "example.com"}
	cfg.applyDefaults()

	if cfg.Port != 22 {
		t.Errorf("expected default port 22, got %d", cfg.Port)
	}
	if cfg.User != "root" {
		t.Errorf("expected default user %q, got %q", "root", cfg.User)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}
}

func TestConfigPreservesExplicitValues(t *testing.T) {
	cfg := Config{
		Host:    "example.com",
		Port:    2222,
		User:    "admin",
		Timeout: 10 * time.Second,
	}
	cfg.applyDefaults()

	if cfg.Port != 2222 {
		t.Errorf("expected port 2222, got %d", cfg.Port)
	}
	if cfg.User != "admin" {
		t.Errorf("expected user %q, got %q", "admin", cfg.User)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", cfg.Timeout)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	c := New(Config{Host: "example.com"})
	if c.config.Port != 22 {
		t.Errorf("New should apply default port, got %d", c.config.Port)
	}
	if c.config.User != "root" {
		t.Errorf("New should apply default user, got %q", c.config.User)
	}
	if c.config.Timeout != 30*time.Second {
		t.Errorf("New should apply default timeout, got %v", c.config.Timeout)
	}
}

func TestNewDoesNotConnect(t *testing.T) {
	c := New(Config{Host: "example.com"})
	if c.SSHClient() != nil {
		t.Error("New should not establish a connection")
	}
}

func TestBuildAuthMethods_PasswordOnly(t *testing.T) {
	// With default fallback, if ~/.ssh/id_rsa doesn't exist, it's skipped silently.
	// So password-only works when the default key file is missing.
	c := New(Config{
		Host:     "example.com",
		Password: "secret",
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auth) == 0 {
		t.Fatal("expected at least one auth method")
	}
}

func TestBuildAuthMethods_PrivateKeyContent(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)

	c := New(Config{
		Host:              "example.com",
		PrivateKeyContent: string(keyPEM),
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auth) == 0 {
		t.Fatal("expected at least one auth method from key content")
	}
}

func TestBuildAuthMethods_PrivateKeyFile(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	c := New(Config{
		Host:       "example.com",
		PrivateKey: keyPath,
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auth) == 0 {
		t.Fatal("expected at least one auth method from key file")
	}
}

func TestBuildAuthMethods_PrivateKeyContentPriority(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)

	// Set both content and a non-existent path. Content should take priority
	// and the path should never be accessed.
	c := New(Config{
		Host:              "example.com",
		PrivateKeyContent: string(keyPEM),
		PrivateKey:        "/nonexistent/should/not/be/read",
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auth) == 0 {
		t.Fatal("expected auth method from key content")
	}
}

func TestBuildAuthMethods_ExplicitKeyFileNotFound(t *testing.T) {
	c := New(Config{
		Host:       "example.com",
		PrivateKey: "/nonexistent/path/to/key",
	})

	_, err := c.buildAuthMethods()
	if err == nil {
		t.Fatal("expected error for missing explicit key file")
	}
}

func TestBuildAuthMethods_NoAuthAvailable(t *testing.T) {
	// Override defaultPrivateKey to point to a non-existent path.
	origDefault := defaultPrivateKey
	defaultPrivateKey = "/nonexistent/default/key"
	defer func() { defaultPrivateKey = origDefault }()

	c := New(Config{
		Host: "example.com",
		// No password, no key content, no key path.
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("buildAuthMethods should not error for missing default key, got: %v", err)
	}
	if len(auth) != 0 {
		t.Fatalf("expected zero auth methods, got %d", len(auth))
	}
}

func TestConnect_EmptyHost(t *testing.T) {
	c := New(Config{})
	err := c.Connect(t.Context())
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestConnect_NoAuth(t *testing.T) {
	origDefault := defaultPrivateKey
	defaultPrivateKey = "/nonexistent/default/key"
	defer func() { defaultPrivateKey = origDefault }()

	c := New(Config{Host: "example.com"})
	err := c.Connect(t.Context())
	if err == nil {
		t.Fatal("expected error when no auth method available")
	}
}

func TestCloseWithoutConnect(t *testing.T) {
	c := New(Config{Host: "example.com"})
	if err := c.Close(); err != nil {
		t.Fatalf("Close on unconnected client should not error, got: %v", err)
	}
}

func TestBuildAuthMethods_InvalidKeyContent(t *testing.T) {
	c := New(Config{
		Host:              "example.com",
		PrivateKeyContent: "not-a-valid-pem-key",
	})

	_, err := c.buildAuthMethods()
	if err == nil {
		t.Fatal("expected error for invalid key content")
	}
}

func TestBuildAuthMethods_PasswordAndKeyContent(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)

	c := New(Config{
		Host:              "example.com",
		Password:          "secret",
		PrivateKeyContent: string(keyPEM),
	})

	auth, err := c.buildAuthMethods()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have both password and public key methods.
	if len(auth) != 2 {
		t.Fatalf("expected 2 auth methods (password + key), got %d", len(auth))
	}
}

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

func TestBuildAuthMethods_InvalidKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "bad_key")
	if err := os.WriteFile(keyPath, []byte("not-a-valid-key"), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	c := New(Config{
		Host:       "example.com",
		PrivateKey: keyPath,
	})

	_, err := c.buildAuthMethods()
	if err == nil {
		t.Fatal("expected error for invalid key file content")
	}
}
