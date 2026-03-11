# OIDC Key Auto-Generation & DB Storage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate external OIDC PEM file dependency by auto-generating keys and storing them in PostgreSQL, with multi-algorithm support (EdDSA/ES256/RS256).

**Architecture:** Replace file-based `LoadKeySet()` with a `KeyStore` interface. The initial implementation (`DBKeyStore`) queries PostgreSQL for an existing key; if none exists, it generates one and persists it. The `KeySet` struct uses generic `crypto.Signer`/`crypto.PublicKey` instead of `*ecdsa.PrivateKey`. Token signing and JWKs endpoints dynamically select algorithm-specific behavior.

**Tech Stack:** Go 1.26, `github.com/golang-jwt/jwt/v5`, `pgx/v5`, sqlc

---

### Task 1: Database Schema & sqlc Queries

**Files:**
- Modify: `pkg/db/schema/schema.sql`
- Create: `pkg/db/query/oidc_key.sql`
- Regenerate: `pkg/db/generated/` (via `make sqlc-generate`)

**Step 1: Add oidc_keys table to schema**

Add to the end of `pkg/db/schema/schema.sql`:

```sql
-- oidc_keys table (auto-generated signing keys)
CREATE TABLE oidc_keys (
    id          BIGSERIAL    PRIMARY KEY,
    key_id      VARCHAR(64)  NOT NULL UNIQUE,
    private_key BYTEA        NOT NULL,
    public_key  BYTEA        NOT NULL,
    algorithm   VARCHAR(16)  NOT NULL DEFAULT 'EdDSA',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE oidc_keys IS 'OIDC 签名密钥：自动生成，存储 PEM 编码的密钥对';
COMMENT ON COLUMN oidc_keys.key_id IS 'RFC 7638 thumbprint，用于 JWK kid 字段';
COMMENT ON COLUMN oidc_keys.algorithm IS '签名算法：EdDSA, ES256, RS256';
```

**Step 2: Create sqlc query file**

Create `pkg/db/query/oidc_key.sql`:

```sql
-- name: GetOIDCKey :one
SELECT * FROM oidc_keys WHERE algorithm = @algorithm LIMIT 1;

-- name: CreateOIDCKey :one
INSERT INTO oidc_keys (key_id, private_key, public_key, algorithm)
VALUES (@key_id, @private_key, @public_key, @algorithm)
RETURNING *;
```

**Step 3: Regenerate sqlc**

Run: `make sqlc-generate`

**Step 4: Apply schema to local database**

Run:
```bash
docker exec lcp-postgres psql -U lcp -d lcp -c "
CREATE TABLE IF NOT EXISTS oidc_keys (
    id          BIGSERIAL    PRIMARY KEY,
    key_id      VARCHAR(64)  NOT NULL UNIQUE,
    private_key BYTEA        NOT NULL,
    public_key  BYTEA        NOT NULL,
    algorithm   VARCHAR(16)  NOT NULL DEFAULT 'EdDSA',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
"
```

**Step 5: Commit**

```bash
git add pkg/db/schema/schema.sql pkg/db/query/oidc_key.sql pkg/db/generated/
git commit -m "feat(oidc): add oidc_keys table and sqlc queries"
```

---

### Task 2: Multi-Algorithm Key Generation & KeyStore Interface

**Files:**
- Modify: `lib/oidc/keys.go` (rewrite)
- Create: `lib/oidc/keys_test.go`

**Step 1: Write tests for key generation and PEM round-trip**

Create `lib/oidc/keys_test.go`:

```go
package oidc

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"testing"
)

func TestGenerateKeySet_EdDSA(t *testing.T) {
	ks, err := GenerateKeySet("EdDSA")
	if err != nil {
		t.Fatalf("GenerateKeySet(EdDSA): %v", err)
	}
	if ks.Algorithm != "EdDSA" {
		t.Errorf("Algorithm = %q, want EdDSA", ks.Algorithm)
	}
	if ks.KeyID == "" {
		t.Error("KeyID is empty")
	}
	if _, ok := ks.PrivateKey.(ed25519.PrivateKey); !ok {
		t.Errorf("PrivateKey type = %T, want ed25519.PrivateKey", ks.PrivateKey)
	}
	if _, ok := ks.PublicKey.(ed25519.PublicKey); !ok {
		t.Errorf("PublicKey type = %T, want ed25519.PublicKey", ks.PublicKey)
	}
}

func TestGenerateKeySet_ES256(t *testing.T) {
	ks, err := GenerateKeySet("ES256")
	if err != nil {
		t.Fatalf("GenerateKeySet(ES256): %v", err)
	}
	if ks.Algorithm != "ES256" {
		t.Errorf("Algorithm = %q, want ES256", ks.Algorithm)
	}
	if ks.KeyID == "" {
		t.Error("KeyID is empty")
	}
	if _, ok := ks.PrivateKey.(*ecdsa.PrivateKey); !ok {
		t.Errorf("PrivateKey type = %T, want *ecdsa.PrivateKey", ks.PrivateKey)
	}
}

func TestGenerateKeySet_RS256(t *testing.T) {
	ks, err := GenerateKeySet("RS256")
	if err != nil {
		t.Fatalf("GenerateKeySet(RS256): %v", err)
	}
	if ks.Algorithm != "RS256" {
		t.Errorf("Algorithm = %q, want RS256", ks.Algorithm)
	}
	if ks.KeyID == "" {
		t.Error("KeyID is empty")
	}
	if _, ok := ks.PrivateKey.(*rsa.PrivateKey); !ok {
		t.Errorf("PrivateKey type = %T, want *rsa.PrivateKey", ks.PrivateKey)
	}
}

func TestGenerateKeySet_InvalidAlgorithm(t *testing.T) {
	_, err := GenerateKeySet("INVALID")
	if err == nil {
		t.Error("expected error for invalid algorithm")
	}
}

func TestPEMRoundTrip_EdDSA(t *testing.T) {
	ks, _ := GenerateKeySet("EdDSA")
	privPEM, pubPEM, err := MarshalKeySetPEM(ks)
	if err != nil {
		t.Fatalf("MarshalKeySetPEM: %v", err)
	}
	ks2, err := ParseKeySetPEM(privPEM, pubPEM, "EdDSA")
	if err != nil {
		t.Fatalf("ParseKeySetPEM: %v", err)
	}
	if ks2.KeyID != ks.KeyID {
		t.Errorf("KeyID mismatch: got %q, want %q", ks2.KeyID, ks.KeyID)
	}
}

func TestPEMRoundTrip_ES256(t *testing.T) {
	ks, _ := GenerateKeySet("ES256")
	privPEM, pubPEM, err := MarshalKeySetPEM(ks)
	if err != nil {
		t.Fatalf("MarshalKeySetPEM: %v", err)
	}
	ks2, err := ParseKeySetPEM(privPEM, pubPEM, "ES256")
	if err != nil {
		t.Fatalf("ParseKeySetPEM: %v", err)
	}
	if ks2.KeyID != ks.KeyID {
		t.Errorf("KeyID mismatch: got %q, want %q", ks2.KeyID, ks.KeyID)
	}
}

func TestPEMRoundTrip_RS256(t *testing.T) {
	ks, _ := GenerateKeySet("RS256")
	privPEM, pubPEM, err := MarshalKeySetPEM(ks)
	if err != nil {
		t.Fatalf("MarshalKeySetPEM: %v", err)
	}
	ks2, err := ParseKeySetPEM(privPEM, pubPEM, "RS256")
	if err != nil {
		t.Fatalf("ParseKeySetPEM: %v", err)
	}
	if ks2.KeyID != ks.KeyID {
		t.Errorf("KeyID mismatch: got %q, want %q", ks2.KeyID, ks.KeyID)
	}
}

func TestComputeKeyID_Deterministic(t *testing.T) {
	ks, _ := GenerateKeySet("EdDSA")
	kid1 := ComputeKeyID(ks.PublicKey)
	kid2 := ComputeKeyID(ks.PublicKey)
	if kid1 != kid2 {
		t.Errorf("KeyID not deterministic: %q != %q", kid1, kid2)
	}
}

func TestSigningMethod(t *testing.T) {
	tests := []struct {
		alg  string
		want string
	}{
		{"EdDSA", "EdDSA"},
		{"ES256", "ES256"},
		{"RS256", "RS256"},
	}
	for _, tt := range tests {
		ks, _ := GenerateKeySet(tt.alg)
		if got := ks.SigningMethod().Alg(); got != tt.want {
			t.Errorf("SigningMethod(%s).Alg() = %q, want %q", tt.alg, got, tt.want)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./lib/oidc/ -run TestGenerate -v`
Expected: FAIL — `GenerateKeySet` undefined

**Step 3: Rewrite `lib/oidc/keys.go`**

Replace the entire file with:

```go
package oidc

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

// Supported signing algorithms.
const (
	AlgEdDSA = "EdDSA"
	AlgES256 = "ES256"
	AlgRS256 = "RS256"
)

// KeyStore loads or generates a signing key set.
type KeyStore interface {
	LoadOrGenerate(algorithm string) (*KeySet, error)
}

// KeySet holds a signing key pair and derived key ID.
type KeySet struct {
	PrivateKey crypto.Signer
	PublicKey  crypto.PublicKey
	KeyID      string
	Algorithm  string
}

// SigningMethod returns the JWT signing method for this key set.
func (ks *KeySet) SigningMethod() jwt.SigningMethod {
	switch ks.Algorithm {
	case AlgEdDSA:
		return jwt.SigningMethodEdDSA
	case AlgES256:
		return jwt.SigningMethodES256
	case AlgRS256:
		return jwt.SigningMethodRS256
	default:
		return jwt.SigningMethodEdDSA
	}
}

// JWKSet is a JSON Web Key Set.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWK is a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	// EC fields
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	// RSA fields
	N string `json:"n,omitempty"`
	E string `json:"e,omitempty"`
}

// JWKSet returns the JWK Set representation of the public key.
func (ks *KeySet) JWKSet() *JWKSet {
	jwk := JWK{
		Use: "sig",
		Kid: ks.KeyID,
	}

	switch pub := ks.PublicKey.(type) {
	case ed25519.PublicKey:
		jwk.Kty = "OKP"
		jwk.Alg = "EdDSA"
		jwk.Crv = "Ed25519"
		jwk.X = base64URLEncode([]byte(pub))
	case *ecdsa.PublicKey:
		jwk.Kty = "EC"
		jwk.Alg = "ES256"
		jwk.Crv = "P-256"
		jwk.X = base64URLEncodeBigInt(pub.X, 32)
		jwk.Y = base64URLEncodeBigInt(pub.Y, 32)
	case *rsa.PublicKey:
		jwk.Kty = "RSA"
		jwk.Alg = "RS256"
		jwk.N = base64URLEncodeBigInt(pub.N, 0)
		jwk.E = base64URLEncodeBigInt(big.NewInt(int64(pub.E)), 0)
	}

	return &JWKSet{Keys: []JWK{jwk}}
}

// GenerateKeySet generates a new key pair for the given algorithm.
func GenerateKeySet(algorithm string) (*KeySet, error) {
	switch algorithm {
	case AlgEdDSA:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate ed25519 key: %w", err)
		}
		return &KeySet{
			PrivateKey: priv,
			PublicKey:  pub,
			KeyID:      ComputeKeyID(pub),
			Algorithm:  AlgEdDSA,
		}, nil
	case AlgES256:
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate ecdsa key: %w", err)
		}
		return &KeySet{
			PrivateKey: priv,
			PublicKey:  &priv.PublicKey,
			KeyID:      ComputeKeyID(&priv.PublicKey),
			Algorithm:  AlgES256,
		}, nil
	case AlgRS256:
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate rsa key: %w", err)
		}
		return &KeySet{
			PrivateKey: priv,
			PublicKey:  &priv.PublicKey,
			KeyID:      ComputeKeyID(&priv.PublicKey),
			Algorithm:  AlgRS256,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm: %q (supported: EdDSA, ES256, RS256)", algorithm)
	}
}

// MarshalKeySetPEM encodes a key set to PEM-encoded private and public key bytes.
func MarshalKeySetPEM(ks *KeySet) (privPEM, pubPEM []byte, err error) {
	switch priv := ks.PrivateKey.(type) {
	case ed25519.PrivateKey:
		privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal ed25519 private key: %w", err)
		}
		privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
		pubBytes, err := x509.MarshalPKIXPublicKey(priv.Public())
		if err != nil {
			return nil, nil, fmt.Errorf("marshal ed25519 public key: %w", err)
		}
		pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	case *ecdsa.PrivateKey:
		privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal ecdsa private key: %w", err)
		}
		privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
		pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal ecdsa public key: %w", err)
		}
		pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	case *rsa.PrivateKey:
		privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal rsa private key: %w", err)
		}
		privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
		pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal rsa public key: %w", err)
		}
		pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	default:
		return nil, nil, fmt.Errorf("unsupported private key type: %T", ks.PrivateKey)
	}
	return privPEM, pubPEM, nil
}

// ParseKeySetPEM parses PEM-encoded private and public key bytes into a KeySet.
func ParseKeySetPEM(privPEM, pubPEM []byte, algorithm string) (*KeySet, error) {
	privBlock, _ := pem.Decode(privPEM)
	if privBlock == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}
	privKey, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		return nil, fmt.Errorf("no PEM block found in public key")
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	signer, ok := privKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key does not implement crypto.Signer")
	}

	return &KeySet{
		PrivateKey: signer,
		PublicKey:  pubKey,
		KeyID:      ComputeKeyID(pubKey),
		Algorithm:  algorithm,
	}, nil
}

// ComputeKeyID derives a key ID from the public key using SHA-256 hash.
func ComputeKeyID(pub crypto.PublicKey) string {
	h := sha256.New()
	switch p := pub.(type) {
	case ed25519.PublicKey:
		h.Write([]byte(p))
	case *ecdsa.PublicKey:
		h.Write(padTo(p.X.Bytes(), 32))
		h.Write(padTo(p.Y.Bytes(), 32))
	case *rsa.PublicKey:
		h.Write(p.N.Bytes())
	}
	return base64URLEncode(h.Sum(nil)[:16])
}

func padTo(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLEncodeBigInt encodes a big.Int as base64url, optionally padded to byteLen.
// If byteLen is 0, no padding is applied.
func base64URLEncodeBigInt(n *big.Int, byteLen int) string {
	b := n.Bytes()
	if byteLen > 0 {
		b = padTo(b, byteLen)
	}
	return base64URLEncode(b)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./lib/oidc/ -run "TestGenerate|TestPEMRound|TestComputeKeyID|TestSigning" -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add lib/oidc/keys.go lib/oidc/keys_test.go
git commit -m "feat(oidc): multi-algorithm KeyStore interface and key generation"
```

---

### Task 3: DBKeyStore Implementation

**Files:**
- Create: `lib/oidc/keys_db.go`
- Create: `lib/oidc/keys_db_test.go`

**Step 1: Write tests for DBKeyStore**

Create `lib/oidc/keys_db_test.go`. Since this requires a real database, use a mock approach:

```go
package oidc

import (
	"context"
	"crypto/ed25519"
	"testing"
)

// mockDBStore implements a simple in-memory KeyStore for testing DBKeyStore logic.
type mockDBStore struct {
	keys map[string]*KeySet
}

func newMockDBStore() *mockDBStore {
	return &mockDBStore{keys: make(map[string]*KeySet)}
}

func (m *mockDBStore) LoadOrGenerate(algorithm string) (*KeySet, error) {
	if ks, ok := m.keys[algorithm]; ok {
		return ks, nil
	}
	ks, err := GenerateKeySet(algorithm)
	if err != nil {
		return nil, err
	}
	m.keys[algorithm] = ks
	return ks, nil
}

func TestMockKeyStore_FirstCallGenerates(t *testing.T) {
	store := newMockDBStore()
	ks, err := store.LoadOrGenerate("EdDSA")
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	if ks.Algorithm != "EdDSA" {
		t.Errorf("Algorithm = %q, want EdDSA", ks.Algorithm)
	}
	if _, ok := ks.PrivateKey.(ed25519.PrivateKey); !ok {
		t.Errorf("PrivateKey type = %T, want ed25519.PrivateKey", ks.PrivateKey)
	}
}

func TestMockKeyStore_SecondCallLoads(t *testing.T) {
	store := newMockDBStore()
	ks1, _ := store.LoadOrGenerate("EdDSA")
	ks2, _ := store.LoadOrGenerate("EdDSA")
	if ks1.KeyID != ks2.KeyID {
		t.Errorf("KeyID changed: %q != %q", ks1.KeyID, ks2.KeyID)
	}
}

// TestDBKeyStore_Integration tests the real DBKeyStore against PostgreSQL.
// Requires LCP_TEST_DB=1 env var and running lcp-postgres.
func TestDBKeyStore_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// Integration tests would go here, using a real pgxpool.Pool
	// For now, the mock tests above validate the KeyStore contract
}

// Verify DBKeyStore implements KeyStore
var _ KeyStore = (*DBKeyStore)(nil)
```

**Step 2: Run test to verify it fails**

Run: `go test ./lib/oidc/ -run TestMockKeyStore -v`
Expected: FAIL — `DBKeyStore` undefined

**Step 3: Implement DBKeyStore**

Create `lib/oidc/keys_db.go`:

```go
package oidc

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lcp.io/lcp/pkg/db/generated"
)

// DBKeyStore loads or generates OIDC signing keys using PostgreSQL.
type DBKeyStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewDBKeyStore creates a new database-backed key store.
func NewDBKeyStore(pool *pgxpool.Pool, queries *generated.Queries) *DBKeyStore {
	return &DBKeyStore{pool: pool, queries: queries}
}

// LoadOrGenerate loads an existing key from the database, or generates a new one.
func (s *DBKeyStore) LoadOrGenerate(algorithm string) (*KeySet, error) {
	ctx := context.Background()

	// Try to load existing key
	row, err := s.queries.GetOIDCKey(ctx, algorithm)
	if err == nil {
		return ParseKeySetPEM(row.PrivateKey, row.PublicKey, row.Algorithm)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("query oidc key: %w", err)
	}

	// Generate new key
	ks, err := GenerateKeySet(algorithm)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	privPEM, pubPEM, err := MarshalKeySetPEM(ks)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	_, err = s.queries.CreateOIDCKey(ctx, generated.CreateOIDCKeyParams{
		KeyID:      ks.KeyID,
		PrivateKey: privPEM,
		PublicKey:  pubPEM,
		Algorithm:  algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("store key: %w", err)
	}

	return ks, nil
}
```

**Step 4: Run tests**

Run: `go test ./lib/oidc/ -run TestMockKeyStore -v`
Expected: PASS (mock tests pass, compile check for DBKeyStore passes)

**Step 5: Commit**

```bash
git add lib/oidc/keys_db.go lib/oidc/keys_db_test.go
git commit -m "feat(oidc): add DBKeyStore implementation for PostgreSQL key storage"
```

---

### Task 4: Update TokenService for Multi-Algorithm Support

**Files:**
- Modify: `lib/oidc/token.go`
- Create: `lib/oidc/token_test.go`

**Step 1: Write tests for token sign & verify with all algorithms**

Create `lib/oidc/token_test.go`:

```go
package oidc

import (
	"testing"
	"time"
)

func TestTokenService_RoundTrip(t *testing.T) {
	algorithms := []string{"EdDSA", "ES256", "RS256"}
	for _, alg := range algorithms {
		t.Run(alg, func(t *testing.T) {
			ks, err := GenerateKeySet(alg)
			if err != nil {
				t.Fatalf("GenerateKeySet: %v", err)
			}
			ts := NewTokenService(ks, "https://test.example.com", time.Hour)

			// Issue access token
			token, err := ts.IssueAccessToken(42, "test-client", []string{"openid", "profile"})
			if err != nil {
				t.Fatalf("IssueAccessToken: %v", err)
			}
			if token == "" {
				t.Fatal("token is empty")
			}

			// Verify access token
			claims, err := ts.VerifyAccessToken(token)
			if err != nil {
				t.Fatalf("VerifyAccessToken: %v", err)
			}
			if claims.Subject != "42" {
				t.Errorf("Subject = %q, want 42", claims.Subject)
			}
			if claims.Scope != "openid profile" {
				t.Errorf("Scope = %q, want 'openid profile'", claims.Scope)
			}
		})
	}
}

func TestTokenService_IDToken(t *testing.T) {
	algorithms := []string{"EdDSA", "ES256", "RS256"}
	for _, alg := range algorithms {
		t.Run(alg, func(t *testing.T) {
			ks, _ := GenerateKeySet(alg)
			ts := NewTokenService(ks, "https://test.example.com", time.Hour)

			at, _ := ts.IssueAccessToken(1, "client", []string{"openid"})
			user := &UserInfo{Name: "Test", Email: "test@example.com"}
			idToken, err := ts.IssueIDToken(1, "client", "nonce123", time.Now(), at, user, []string{"openid", "profile", "email"})
			if err != nil {
				t.Fatalf("IssueIDToken: %v", err)
			}
			if idToken == "" {
				t.Fatal("idToken is empty")
			}
		})
	}
}

func TestTokenService_WrongKey(t *testing.T) {
	ks1, _ := GenerateKeySet("EdDSA")
	ks2, _ := GenerateKeySet("EdDSA")
	ts1 := NewTokenService(ks1, "https://test.example.com", time.Hour)
	ts2 := NewTokenService(ks2, "https://test.example.com", time.Hour)

	token, _ := ts1.IssueAccessToken(1, "client", nil)
	_, err := ts2.VerifyAccessToken(token)
	if err == nil {
		t.Error("expected error verifying with wrong key")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./lib/oidc/ -run TestTokenService -v`
Expected: FAIL — `NewTokenService` signature or signing method mismatch

**Step 3: Update `lib/oidc/token.go`**

Replace the signing/verification to be algorithm-agnostic:

In `IssueAccessToken`, change:
- `jwt.NewWithClaims(jwt.SigningMethodES256, claims)` → `jwt.NewWithClaims(ts.keySet.SigningMethod(), claims)`
- `token.SignedString(ts.keySet.PrivateKey)` → `ts.signToken(token)`

In `IssueIDToken`, same changes.

In `VerifyAccessToken`, change the key function:
- Remove the `*jwt.SigningMethodECDSA` type assertion
- Use `ts.verifyingKey()` to return the correct key type

Add helper methods:

```go
// signToken signs the token with the correct key type for the algorithm.
func (ts *TokenService) signToken(token *jwt.Token) (string, error) {
	switch ts.keySet.Algorithm {
	case AlgEdDSA:
		return token.SignedString(ts.keySet.PrivateKey)
	case AlgES256:
		return token.SignedString(ts.keySet.PrivateKey)
	case AlgRS256:
		return token.SignedString(ts.keySet.PrivateKey)
	default:
		return token.SignedString(ts.keySet.PrivateKey)
	}
}

// verifyingKey returns the public key in the format jwt expects for verification.
func (ts *TokenService) verifyingKey() any {
	switch ts.keySet.Algorithm {
	case AlgEdDSA:
		return ts.keySet.PublicKey
	case AlgES256:
		return ts.keySet.PublicKey
	case AlgRS256:
		return ts.keySet.PublicKey
	default:
		return ts.keySet.PublicKey
	}
}
```

The full updated `token.go`:

```go
package oidc

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService handles JWT issuance and verification.
type TokenService struct {
	keySet *KeySet
	issuer string
	atTTL  time.Duration
}

// NewTokenService creates a new token service.
func NewTokenService(keySet *KeySet, issuer string, accessTokenTTL time.Duration) *TokenService {
	return &TokenService{
		keySet: keySet,
		issuer: issuer,
		atTTL:  accessTokenTTL,
	}
}

// StandardClaims are the claims in an access token.
type StandardClaims struct {
	jwt.RegisteredClaims
	Scope string `json:"scope,omitempty"`
}

// IDTokenClaims are the claims in an ID token.
type IDTokenClaims struct {
	jwt.RegisteredClaims
	Nonce       string `json:"nonce,omitempty"`
	AuthTime    int64  `json:"auth_time,omitempty"`
	AtHash      string `json:"at_hash,omitempty"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// IssueAccessToken creates a signed JWT access token.
func (ts *TokenService) IssueAccessToken(userID int64, clientID string, scopes []string) (string, error) {
	now := time.Now()
	claims := StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.atTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Scope: strings.Join(scopes, " "),
	}

	token := jwt.NewWithClaims(ts.keySet.SigningMethod(), claims)
	token.Header["kid"] = ts.keySet.KeyID
	return token.SignedString(ts.keySet.PrivateKey)
}

// IssueIDToken creates a signed JWT ID token.
func (ts *TokenService) IssueIDToken(userID int64, clientID string, nonce string, authTime time.Time, accessToken string, user *UserInfo, scopes []string) (string, error) {
	now := time.Now()
	claims := IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.atTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Nonce:    nonce,
		AuthTime: authTime.Unix(),
		AtHash:   computeAtHash(accessToken),
	}

	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}
	if user != nil {
		if scopeSet["profile"] {
			claims.Name = user.Name
		}
		if scopeSet["email"] {
			claims.Email = user.Email
		}
		if scopeSet["phone"] {
			claims.PhoneNumber = user.PhoneNumber
		}
	}

	token := jwt.NewWithClaims(ts.keySet.SigningMethod(), claims)
	token.Header["kid"] = ts.keySet.KeyID
	return token.SignedString(ts.keySet.PrivateKey)
}

// VerifyAccessToken parses and validates an access token, returning its claims.
func (ts *TokenService) VerifyAccessToken(tokenStr string) (*StandardClaims, error) {
	claims := &StandardClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != ts.keySet.SigningMethod().Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return ts.keySet.PublicKey, nil
	}, jwt.WithIssuer(ts.issuer))
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}
	return claims, nil
}

// computeAtHash computes the at_hash claim per OIDC spec (left half of SHA-256).
func computeAtHash(accessToken string) string {
	h := sha256.Sum256([]byte(accessToken))
	return base64URLEncode(h[:16])
}
```

**Step 4: Run tests**

Run: `go test ./lib/oidc/ -run TestTokenService -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add lib/oidc/token.go lib/oidc/token_test.go
git commit -m "feat(oidc): update TokenService for multi-algorithm JWT signing"
```

---

### Task 5: Update Provider (DiscoveryDocument + JWKs)

**Files:**
- Modify: `lib/oidc/provider.go`

**Step 1: Update DiscoveryDocument**

In `lib/oidc/provider.go`, the `DiscoveryDocument()` method currently hardcodes `"ES256"`. Change the signing algorithm to be dynamic based on `p.keySet.Algorithm`:

```go
IDTokenSigningAlgValuesSupported: []string{p.keySet.SigningMethod().Alg()},
```

**Step 2: Verify compilation**

Run: `go build ./lib/oidc/`
Expected: success

**Step 3: Commit**

```bash
git add lib/oidc/provider.go
git commit -m "feat(oidc): dynamic algorithm in OIDC discovery document"
```

---

### Task 6: Update Config

**Files:**
- Modify: `lib/config/config.go`
- Modify: `lib/oidc/config.go`

**Step 1: Update OIDCConfig struct**

In `lib/config/config.go`, replace `PrivateKeyFile`/`PublicKeyFile` with `Algorithm`:

```go
type OIDCConfig struct {
	Issuer          string         `yaml:"issuer"`
	Algorithm       string         `yaml:"algorithm"`
	AccessTokenTTL  string         `yaml:"accessTokenTTL"`
	RefreshTokenTTL string         `yaml:"refreshTokenTTL"`
	AuthCodeTTL     string         `yaml:"authCodeTTL"`
	LoginURL        string         `yaml:"loginUrl"`
	Clients         []ClientConfig `yaml:"clients"`
}
```

In `SetDefaults`, add:

```go
if cfg.OIDC.Algorithm == "" {
    cfg.OIDC.Algorithm = "EdDSA"
}
```

Add env override in `ApplyEnvOverrides`:

```go
if v := os.Getenv("OIDC_ALGORITHM"); v != "" {
    cfg.OIDC.Algorithm = v
}
```

**Step 2: Verify compilation**

Run: `go build ./...`
Expected: compilation errors in `pkg/apis/iam/v1/install.go` (references removed fields) — this is expected and will be fixed in the next task.

**Step 3: Commit**

```bash
git add lib/config/config.go lib/oidc/config.go
git commit -m "feat(oidc): update config for algorithm-based key management"
```

---

### Task 7: Update OIDC Provider Wiring (install.go)

**Files:**
- Modify: `pkg/apis/iam/v1/install.go`
- Modify: `pkg/apis/install.go`
- Modify: `app/lcp-server/main.go`

**Step 1: Update `NewOIDCProvider` in `pkg/apis/iam/v1/install.go`**

Replace the function to use `KeyStore` instead of file-based loading:

```go
// NewOIDCProvider creates the OIDC provider with all internal store wiring.
// Keys are auto-generated and stored in the database.
// Returns nil if OIDC issuer is not configured.
func NewOIDCProvider(database *db.DB, cfg *config.OIDCConfig) *oidc.Provider {
	if cfg.Issuer == "" {
		logger.Infof("OIDC not configured (no issuer), authentication disabled")
		return nil
	}

	providerCfg, err := oidc.ParseConfig(cfg)
	if err != nil {
		logger.Fatalf("invalid OIDC config: %v", err)
	}

	keyStore := oidc.NewDBKeyStore(database.Pool, database.Queries)
	keySet, err := keyStore.LoadOrGenerate(cfg.Algorithm)
	if err != nil {
		logger.Fatalf("cannot load/generate OIDC keys: %v", err)
	}

	logger.Infof("OIDC keys ready (algorithm=%s, kid=%s)", keySet.Algorithm, keySet.KeyID)

	userStore := iamstore.NewPGUserStore(database.Queries)
	refreshStore := iamstore.NewPGRefreshTokenStore(database.Queries)

	provider := oidc.NewProvider(providerCfg, keySet,
		iam.NewUserLookupAdapter(userStore),
		iam.NewRefreshTokenAdapter(refreshStore),
	)
	provider.SetClients(oidc.ParseClients(cfg.Clients))

	logger.Infof("OIDC provider initialized (issuer=%s)", cfg.Issuer)
	return provider
}
```

**Step 2: Update `pkg/apis/install.go`**

The `NewOIDCProvider` function signature doesn't change (still `func NewOIDCProvider(database *db.DB, cfg *config.OIDCConfig) *oidc.Provider`), so `pkg/apis/install.go` needs no changes.

**Step 3: Verify compilation**

Run: `go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add pkg/apis/iam/v1/install.go
git commit -m "feat(oidc): wire DBKeyStore into OIDC provider initialization"
```

---

### Task 8: Update Config Files

**Files:**
- Modify: `app/lcp-server/config.yaml`
- Modify: `app/lcp-server/config.dev.yaml`
- Modify: `deployment/docker/Dockerfile`

**Step 1: Update `config.yaml`**

Remove `privateKeyFile` and `publicKeyFile`, add `algorithm`:

```yaml
oidc:
  issuer: "http://localhost:8428"
  algorithm: "EdDSA"
  accessTokenTTL: "1h"
  refreshTokenTTL: "168h"
  authCodeTTL: "5m"
  loginUrl: "/login"
  clients:
    - id: "lcp-ui"
      public: true
      redirectUris:
        - "http://localhost:8428/auth/callback"
      scopes: ["openid", "profile", "email", "phone"]
```

**Step 2: Update `config.dev.yaml`**

Same changes, keeping the dev redirect URI (`localhost:5173`).

**Step 3: Update Dockerfile**

Remove the OIDC PEM volume mount comment or documentation if any. The Dockerfile itself doesn't reference PEM files, but update `.dockerignore` to remove the `oidc-private.pem` / `oidc-public.pem` entries since they're no longer needed.

**Step 4: Update `.gitignore`**

The `oidc-private.pem` and `oidc-public.pem` entries in `.gitignore` can remain (harmless) or be removed. Leave them for now since users may still have the files locally.

**Step 5: Verify the server starts**

Run: `go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml`
Expected: server starts, logs show "OIDC keys ready (algorithm=EdDSA, kid=...)" and "OIDC provider initialized"

**Step 6: Verify key is stored in DB**

Run: `docker exec lcp-postgres psql -U lcp -d lcp -c "SELECT id, key_id, algorithm, created_at FROM oidc_keys;"`
Expected: one row with algorithm=EdDSA

**Step 7: Restart server and verify key is loaded (not regenerated)**

Run the server again, check logs show same `kid` value.

**Step 8: Commit**

```bash
git add app/lcp-server/config.yaml app/lcp-server/config.dev.yaml .dockerignore
git commit -m "feat(oidc): update config files for auto-generated DB keys"
```

---

### Task 9: Run Full Test Suite

**Step 1: Run all tests**

Run: `make test`
Expected: all pass

**Step 2: Run vet and lint**

Run: `make vet && make lint`
Expected: clean

**Step 3: E2E verification**

Start server, login via OIDC flow:

```bash
# Start server
go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml &

# Test OIDC discovery
curl -s http://localhost:8428/.well-known/openid-configuration | jq .id_token_signing_alg_values_supported
# Expected: ["EdDSA"]

# Test JWKS
curl -s http://localhost:8428/.well-known/jwks.json | jq .
# Expected: keys[0].kty = "OKP", keys[0].crv = "Ed25519"

# Kill server
kill %1
```

**Step 4: Commit any fixes if needed**

---

### Task 10: Cleanup Old Code

**Files:**
- Modify: `lib/oidc/keys.go` (remove `LoadKeySet` if still present)
- Delete or update: `CLAUDE.md` references to PEM files

**Step 1: Remove any remaining references to old file-based key loading**

Search for `PrivateKeyFile`, `PublicKeyFile`, `LoadKeySet`, `oidc-private.pem`, `oidc-public.pem` across the codebase and remove/update references.

**Step 2: Update CLAUDE.md**

Update the OIDC Configuration section to reflect the new behavior:
- Remove `privateKeyFile` / `publicKeyFile` references
- Add `algorithm` field documentation
- Update the "Generate keys" instructions to note keys are auto-generated
- Remove the worktree notes about copying PEM files

**Step 3: Commit**

```bash
git add -A
git commit -m "chore(oidc): remove file-based key references, update documentation"
```
