package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/pki"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgCertificateStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGCertificateStore creates a PostgreSQL-backed CertificateStore.
func NewPGCertificateStore(pool *pgxpool.Pool, queries *generated.Queries) pki.CertificateStore {
	return &pgCertificateStore{db: pool, queries: queries}
}

func (s *pgCertificateStore) Create(ctx context.Context, cert *pki.DBCertificate) (*pki.DBCertificate, error) {
	row, err := s.queries.CreateCertificate(ctx, generated.CreateCertificateParams{
		Name:         cert.Name,
		CertType:     cert.CertType,
		CommonName:   cert.CommonName,
		DnsNames:     cert.DnsNames,
		IpAddresses:  cert.IpAddresses,
		CaName:       cert.CaName,
		SerialNumber: cert.SerialNumber,
		Certificate:  cert.Certificate,
		PrivateKey:   cert.PrivateKey,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("certificate", cert.Name)
		}
		return nil, fmt.Errorf("create certificate: %w", err)
	}
	return &row, nil
}

func (s *pgCertificateStore) GetByID(ctx context.Context, id int64) (*pki.DBCertificate, error) {
	row, err := s.queries.GetCertificateByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("certificate", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get certificate: %w", err)
	}
	return &row, nil
}

func (s *pgCertificateStore) GetByName(ctx context.Context, name string) (*pki.DBCertificate, error) {
	row, err := s.queries.GetCertificateByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("certificate", name)
		}
		return nil, fmt.Errorf("get certificate by name: %w", err)
	}
	return &row, nil
}

func (s *pgCertificateStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[pki.DBCertificate], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountCertificates(ctx, generated.CountCertificatesParams{
		CertType: filterStr(q.Filters, "certType"),
		CaName:   filterStr(q.Filters, "caName"),
		Search:   filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count certificates: %w", err)
	}

	rows, err := s.queries.ListCertificates(ctx, generated.ListCertificatesParams{
		CertType:   filterStr(q.Filters, "certType"),
		CaName:     filterStr(q.Filters, "caName"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}

	return &db.ListResult[pki.DBCertificate]{Items: rows, TotalCount: count}, nil
}

func (s *pgCertificateStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteCertificate(ctx, id); err != nil {
		return fmt.Errorf("delete certificate: %w", err)
	}
	return nil
}

func (s *pgCertificateStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// Use a transaction to atomically check CA dependencies and delete
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Check each certificate for CA dependencies before deleting
	for _, id := range ids {
		row, err := qtx.GetCertificateByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // already deleted
			}
			return 0, fmt.Errorf("get certificate %d: %w", id, err)
		}
		if row.CertType == "ca" {
			cn := row.Name
			count, err := qtx.CountCertificatesByCAName(ctx, &cn)
			if err != nil {
				return 0, fmt.Errorf("count dependents of CA %q: %w", row.Name, err)
			}
			if count > 0 {
				return 0, apierrors.NewBadRequest(
					fmt.Sprintf("cannot delete CA %q: %d certificate(s) depend on it", row.Name, count), nil)
			}
		}
	}

	deletedIDs, err := qtx.DeleteCertificates(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete certificates: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return int64(len(deletedIDs)), nil
}

func (s *pgCertificateStore) CountByCAName(ctx context.Context, caName string) (int64, error) {
	cn := caName
	count, err := s.queries.CountCertificatesByCAName(ctx, &cn)
	if err != nil {
		return 0, fmt.Errorf("count certificates by CA name: %w", err)
	}
	return count, nil
}
