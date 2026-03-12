package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/network"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgIPAllocationStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGIPAllocationStore 创建 PostgreSQL 实现的 IPAllocationStore。
func NewPGIPAllocationStore(pool *pgxpool.Pool, queries *generated.Queries) network.IPAllocationStore {
	return &pgIPAllocationStore{pool: pool, queries: queries}
}

func (s *pgIPAllocationStore) Create(ctx context.Context, tx pgx.Tx, alloc *network.DBIPAllocation) (*network.DBIPAllocation, error) {
	q := s.queries
	if tx != nil {
		q = q.WithTx(tx)
	}

	row, err := q.CreateIPAllocation(ctx, generated.CreateIPAllocationParams{
		SubnetID:    alloc.SubnetID,
		Ip:          alloc.Ip,
		Description: alloc.Description,
		IsGateway:   alloc.IsGateway,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("ip_allocation", alloc.Ip)
		}
		return nil, fmt.Errorf("create ip allocation: %w", err)
	}
	return &row, nil
}

func (s *pgIPAllocationStore) GetBySubnetAndIP(ctx context.Context, subnetID int64, ip string) (*network.DBIPAllocation, error) {
	row, err := s.queries.GetIPAllocationBySubnetAndIP(ctx, generated.GetIPAllocationBySubnetAndIPParams{
		SubnetID: subnetID,
		Ip:       ip,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("ip_allocation", ip)
		}
		return nil, fmt.Errorf("get ip allocation: %w", err)
	}
	return &row, nil
}

func (s *pgIPAllocationStore) Delete(ctx context.Context, id int64) error {
	err := s.queries.DeleteIPAllocation(ctx, id)
	if err != nil {
		return fmt.Errorf("delete ip allocation: %w", err)
	}
	return nil
}

func (s *pgIPAllocationStore) DeleteBySubnetID(ctx context.Context, tx pgx.Tx, subnetID int64) error {
	q := s.queries
	if tx != nil {
		q = q.WithTx(tx)
	}

	err := q.DeleteIPAllocationsBySubnetID(ctx, subnetID)
	if err != nil {
		return fmt.Errorf("delete ip allocations by subnet id: %w", err)
	}
	return nil
}

func (s *pgIPAllocationStore) List(ctx context.Context, subnetID int64, q db.ListQuery) (*db.ListResult[network.DBIPAllocation], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountIPAllocations(ctx, generated.CountIPAllocationsParams{
		SubnetID:  subnetID,
		IsGateway: filterBool(q.Filters, "isGateway"),
		Search:    filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count ip allocations: %w", err)
	}

	rows, err := s.queries.ListIPAllocations(ctx, generated.ListIPAllocationsParams{
		SubnetID:  subnetID,
		IsGateway: filterBool(q.Filters, "isGateway"),
		Search:    filterStr(q.Filters, "search"),
		SortField: q.SortBy,
		SortOrder: sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list ip allocations: %w", err)
	}

	return &db.ListResult[network.DBIPAllocation]{Items: rows, TotalCount: count}, nil
}
