package store

import (
	"lcp.io/lcp/pkg/apis/pki"
	"lcp.io/lcp/pkg/db"
)

// NewStores creates all PKI store implementations.
func NewStores(database *db.DB) pki.Stores {
	return pki.Stores{
		Certificate: NewPGCertificateStore(database.Queries),
	}
}
