package pki

import (
	"context"

	"lcp.io/lcp/pkg/db"
)

// CertificateStore defines the data access interface for certificates.
type CertificateStore interface {
	Create(ctx context.Context, cert *DBCertificate) (*DBCertificate, error)
	GetByID(ctx context.Context, id int64) (*DBCertificate, error)
	GetByName(ctx context.Context, name string) (*DBCertificate, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBCertificate], error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	CountByCAName(ctx context.Context, caName string) (int64, error)
}
