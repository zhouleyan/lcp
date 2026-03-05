package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

// KeySet holds the ECDSA key pair and derived key ID.
type KeySet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	KeyID      string
}

// JWKSet is a JSON Web Key Set.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWK is a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

// LoadKeySet loads an ECDSA P-256 key pair from PEM files.
func LoadKeySet(privatePEMPath, publicPEMPath string) (*KeySet, error) {
	privData, err := os.ReadFile(privatePEMPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", privatePEMPath, err)
	}
	pubData, err := os.ReadFile(publicPEMPath)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", publicPEMPath, err)
	}

	privKey, err := parseECPrivateKey(privData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	pubKey, err := parseECPublicKey(pubData)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	if privKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("private key must be ECDSA P-256, got %s", privKey.Curve.Params().Name)
	}

	kid := computeKeyID(pubKey)

	return &KeySet{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		KeyID:      kid,
	}, nil
}

// JWKSet returns the JWK Set representation of the public key.
func (ks *KeySet) JWKSet() *JWKSet {
	return &JWKSet{
		Keys: []JWK{
			{
				Kty: "EC",
				Crv: "P-256",
				X:   base64URLEncodeBigInt(ks.PublicKey.X),
				Y:   base64URLEncodeBigInt(ks.PublicKey.Y),
				Kid: ks.KeyID,
				Use: "sig",
				Alg: "ES256",
			},
		},
	}
}

func parseECPrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	// Try PKCS8 first, then EC private key
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if ecKey, ok := key.(*ecdsa.PrivateKey); ok {
			return ecKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not ECDSA")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

func parseECPublicKey(data []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	return ecPub, nil
}

// computeKeyID derives a key ID from the public key using RFC 7638 thumbprint.
func computeKeyID(pub *ecdsa.PublicKey) string {
	// Pad X and Y coordinates to 32 bytes for P-256
	x := padTo32(pub.X.Bytes())
	y := padTo32(pub.Y.Bytes())
	h := sha256.New()
	h.Write(x)
	h.Write(y)
	return base64URLEncode(h.Sum(nil)[:16])
}

func padTo32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLEncodeBigInt encodes a big.Int as base64url with padding to byte length.
func base64URLEncodeBigInt(n *big.Int) string {
	return base64URLEncode(padTo32(n.Bytes()))
}
