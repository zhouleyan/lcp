# OIDC Key Management: Auto-Generate & DB Storage

## Background

Current OIDC implementation requires manually generating ECDSA P-256 PEM files and mounting them into the container. This creates deployment friction and security management burden.

## Goal

Eliminate external key file dependency. Keys are auto-generated at startup and persisted in PostgreSQL.

## Design

### Database Table

```sql
CREATE TABLE oidc_keys (
    id          BIGSERIAL    PRIMARY KEY,
    key_id      VARCHAR(64)  NOT NULL UNIQUE,
    private_key BYTEA        NOT NULL,
    public_key  BYTEA        NOT NULL,
    algorithm   VARCHAR(16)  NOT NULL DEFAULT 'EdDSA',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

Single row. `key_id` is RFC 7638 thumbprint for JWKs `kid` field.

### Supported Algorithms

| Algorithm | Key Type | JWT Method | JWK Key Type | Default |
|-----------|----------|------------|-------------|---------|
| EdDSA     | Ed25519  | `jwt.SigningMethodEdDSA` | `OKP` | Yes |
| ES256     | ECDSA P-256 | `jwt.SigningMethodES256` | `EC` | No |
| RS256     | RSA 2048 | `jwt.SigningMethodRS256` | `RSA` | No |

### KeyStore Interface

```go
// lib/oidc/keys.go

type KeyStore interface {
    LoadOrGenerate(algorithm string) (*KeySet, error)
}

type KeySet struct {
    PrivateKey crypto.Signer
    PublicKey  crypto.PublicKey
    KeyID      string
    Algorithm  string  // "EdDSA", "ES256", "RS256"
}

func (ks *KeySet) SigningMethod() jwt.SigningMethod
```

- `DBKeyStore` (PostgreSQL) is the initial implementation.
- Interface allows future implementations (file, Vault, etc.).

### Startup Flow

```
Start
  -> Create KeyStore based on config (default: DBKeyStore)
  -> keyStore.LoadOrGenerate(algorithm)
      -> Query oidc_keys table
      -> Record exists with matching algorithm?
          Yes -> Parse PEM, return KeySet
          No  -> Generate key pair by algorithm -> Store in DB -> Return KeySet
  -> Initialize Provider with KeySet (same as current)
```

### Config Changes

```yaml
oidc:
  issuer: "http://localhost:8428"
  algorithm: "EdDSA"       # Default EdDSA, options: ES256, RS256
  # privateKeyFile / publicKeyFile removed
  accessTokenTTL: "1h"
  refreshTokenTTL: "168h"
  authCodeTTL: "5m"
  loginUrl: "/login"
  clients:
    - id: "lcp-ui"
      public: true
      redirectUris: ["http://localhost:8428/auth/callback"]
      scopes: ["openid", "profile", "email", "phone"]
```

### Code Changes

| File | Change |
|------|--------|
| `pkg/db/schema/schema.sql` | Add `oidc_keys` table |
| `pkg/db/query/oidc_keys.sql` | sqlc queries (get, create) |
| `lib/config/config.go` | Add `Algorithm` field, remove `PrivateKeyFile`/`PublicKeyFile` |
| `lib/oidc/keys.go` | `KeyStore` interface, `KeySet` with generic crypto types, key generation/parsing per algorithm |
| `lib/oidc/keys_db.go` (new) | `DBKeyStore` implementation |
| `lib/oidc/token.go` | Dynamic signing method from `KeySet.Algorithm` |
| `lib/oidc/provider.go` | JWKs endpoint supports OKP, EC, RSA key types |
| `pkg/apis/iam/v1/install.go` | Use `KeyStore` interface instead of `LoadKeySet` |
| `app/lcp-server/config.yaml` | Remove key file paths, add `algorithm` |

### Tests

| File | Coverage |
|------|----------|
| `lib/oidc/keys_test.go` | Key generation, PEM encode/decode, KeyID computation for all 3 algorithms |
| `lib/oidc/keys_db_test.go` | DBKeyStore LoadOrGenerate (first-time generate / subsequent load) |
| `lib/oidc/token_test.go` | Token sign & verify for all 3 algorithms |
