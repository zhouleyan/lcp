package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db/generated"
)

type pgIPBinder struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGIPBinder creates a new PostgreSQL-backed IPBinder.
func NewPGIPBinder(pool *pgxpool.Pool, queries *generated.Queries) infra.IPBinder {
	return &pgIPBinder{pool: pool, queries: queries}
}

func (s *pgIPBinder) GetSubnetForUpdate(ctx context.Context, tx pgx.Tx, subnetID int64) (*infra.DBSubnetRow, error) {
	row, err := s.queries.WithTx(tx).GetSubnetByIDForUpdateACL(ctx, subnetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("subnet", fmt.Sprintf("%d", subnetID))
		}
		return nil, fmt.Errorf("get subnet for update: %w", err)
	}
	return &row, nil
}

func (s *pgIPBinder) UpdateSubnetBitmap(ctx context.Context, tx pgx.Tx, subnetID int64, bitmap []byte) error {
	err := s.queries.WithTx(tx).UpdateSubnetBitmapACL(ctx, generated.UpdateSubnetBitmapACLParams{
		ID:     subnetID,
		Bitmap: bitmap,
	})
	if err != nil {
		return fmt.Errorf("update subnet bitmap: %w", err)
	}
	return nil
}

func (s *pgIPBinder) CreateIPAllocation(ctx context.Context, tx pgx.Tx, alloc *infra.DBIPAllocationWithHost) (*infra.DBIPAllocationWithHost, error) {
	row, err := s.queries.WithTx(tx).CreateIPAllocationWithHost(ctx, generated.CreateIPAllocationWithHostParams{
		SubnetID:    alloc.SubnetID,
		Ip:          alloc.Ip,
		Description: alloc.Description,
		IsGateway:   alloc.IsGateway,
		HostID:      alloc.HostID,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("ip_allocation", alloc.Ip)
		}
		return nil, fmt.Errorf("create ip allocation: %w", err)
	}
	return &row, nil
}

func (s *pgIPBinder) UnbindIPAllocationFromHost(ctx context.Context, allocID, hostID int64) error {
	n, err := s.queries.UnbindIPAllocationFromHost(ctx, generated.UnbindIPAllocationFromHostParams{
		ID:     allocID,
		HostID: &hostID,
	})
	if err != nil {
		return fmt.Errorf("unbind ip allocation: %w", err)
	}
	if n == 0 {
		return apierrors.NewNotFound("ip_allocation", fmt.Sprintf("%d", allocID))
	}
	return nil
}

