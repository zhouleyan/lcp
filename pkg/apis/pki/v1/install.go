package v1

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/pki"
	pkistore "lcp.io/lcp/pkg/apis/pki/store"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// ModuleResult holds the output of PKI module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewPKIModule initializes the PKI module.
func NewPKIModule(database *db.DB) ModuleResult {
	encryptionKey, err := loadOrGenerateEncryptionKey(database.Queries)
	if err != nil {
		logger.Fatalf("cannot load/generate PKI encryption key: %v", err)
	}
	logger.Infof("PKI encryption key ready")

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

// loadOrGenerateEncryptionKey loads the AES-256 encryption key from the database,
// or generates and stores a new one if none exists.
func loadOrGenerateEncryptionKey(queries *generated.Queries) ([]byte, error) {
	ctx := context.Background()

	// Try to load existing key
	row, err := queries.GetPKIEncryptionKey(ctx)
	if err == nil {
		if len(row.EncryptionKey) != 32 {
			return nil, fmt.Errorf("stored encryption key has invalid length %d (expected 32)", len(row.EncryptionKey))
		}
		return row.EncryptionKey, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("query encryption key: %w", err)
	}

	// Generate new 32-byte AES key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	_, err = queries.CreatePKIEncryptionKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("store key: %w", err)
	}

	return key, nil
}
