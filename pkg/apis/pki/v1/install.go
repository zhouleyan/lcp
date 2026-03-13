package v1

import (
	"crypto/rand"
	"encoding/base64"

	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/pki"
	pkistore "lcp.io/lcp/pkg/apis/pki/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of PKI module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewPKIModule initializes the PKI module.
func NewPKIModule(database *db.DB) ModuleResult {
	encryptionKey := resolveEncryptionKey()

	p := pki.NewRESTStorageProvider(pkistore.NewStores(database))
	certStorage := pki.NewCertificateStorage(p.Certificate, encryptionKey)
	exportHandler := pki.NewExportHandler(p.Certificate, encryptionKey)

	group := &rest.APIGroupInfo{
		GroupName: "pki",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "certificates",
				Storage: certStorage,
				Actions: []rest.ActionInfo{
					{
						Name:    "export",
						Method:  "GET",
						Handler: exportHandler,
					},
				},
			},
		},
	}

	return ModuleResult{Group: group}
}

// resolveEncryptionKey returns the 32-byte AES key for private key encryption.
// If not configured, generates a random key (logged as warning).
func resolveEncryptionKey() []byte {
	cfg := config.Get()
	if cfg != nil && cfg.PKI.EncryptionKey != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.PKI.EncryptionKey)
		if err != nil {
			logger.Fatalf("invalid pki.encryptionKey (must be base64): %v", err)
		}
		if len(key) != 32 {
			logger.Fatalf("pki.encryptionKey must be 32 bytes (got %d)", len(key))
		}
		return key
	}

	logger.Warnf("pki.encryptionKey not configured, generating random key (certificates will not survive restart without config)")
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		logger.Fatalf("cannot generate encryption key: %v", err)
	}
	logger.Infof("generated PKI encryption key: %s (add to config.yaml to persist)", base64.StdEncoding.EncodeToString(key))
	return key
}
