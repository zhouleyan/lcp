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

type pgSubnetStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGSubnetStore 创建 PostgreSQL 实现的 SubnetStore。
func NewPGSubnetStore(pool *pgxpool.Pool, queries *generated.Queries) network.SubnetStore {
	return &pgSubnetStore{pool: pool, queries: queries}
}

func (s *pgSubnetStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

func (s *pgSubnetStore) Create(ctx context.Context, tx pgx.Tx, subnet *network.DBSubnet) (*network.DBSubnet, error) {
	q := s.queries
	if tx != nil {
		q = q.WithTx(tx)
	}

	row, err := q.CreateSubnet(ctx, generated.CreateSubnetParams{
		Name:        subnet.Name,
		DisplayName: subnet.DisplayName,
		Description: subnet.Description,
		NetworkID:   subnet.NetworkID,
		Cidr:        subnet.Cidr,
		Gateway:     subnet.Gateway,
		Bitmap:      subnet.Bitmap,
		Status:      subnet.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("subnet", subnet.Name)
		}
		return nil, fmt.Errorf("create subnet: %w", err)
	}
	return &row, nil
}

func (s *pgSubnetStore) GetByID(ctx context.Context, id int64) (*network.DBSubnet, error) {
	row, err := s.queries.GetSubnetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("subnet", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get subnet by id: %w", err)
	}
	return &row, nil
}

func (s *pgSubnetStore) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*network.DBSubnet, error) {
	row, err := s.queries.WithTx(tx).GetSubnetByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("subnet", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get subnet for update: %w", err)
	}
	return &row, nil
}

func (s *pgSubnetStore) Update(ctx context.Context, subnet *network.DBSubnet) (*network.DBSubnet, error) {
	row, err := s.queries.UpdateSubnet(ctx, generated.UpdateSubnetParams{
		ID:          subnet.ID,
		Name:        subnet.Name,
		DisplayName: subnet.DisplayName,
		Description: subnet.Description,
		Status:      subnet.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("subnet", fmt.Sprintf("%d", subnet.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("subnet", subnet.Name)
		}
		return nil, fmt.Errorf("update subnet: %w", err)
	}
	return &row, nil
}

func (s *pgSubnetStore) Patch(ctx context.Context, id int64, fields map[string]any) (*network.DBSubnet, error) {
	params := generated.PatchSubnetParams{ID: id}
	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}

	row, err := s.queries.PatchSubnet(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("subnet", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("subnet", "")
		}
		return nil, fmt.Errorf("patch subnet: %w", err)
	}
	return &row, nil
}

func (s *pgSubnetStore) UpdateBitmap(ctx context.Context, tx pgx.Tx, id int64, bitmap []byte) error {
	err := s.queries.WithTx(tx).UpdateSubnetBitmap(ctx, generated.UpdateSubnetBitmapParams{
		ID:     id,
		Bitmap: bitmap,
	})
	if err != nil {
		return fmt.Errorf("update subnet bitmap: %w", err)
	}
	return nil
}

func (s *pgSubnetStore) Delete(ctx context.Context, id int64) error {
	err := s.queries.DeleteSubnet(ctx, id)
	if err != nil {
		return fmt.Errorf("delete subnet: %w", err)
	}
	return nil
}

func (s *pgSubnetStore) DeleteByIDs(ctx context.Context, networkID int64, ids []int64) (int64, error) {
	deleted, err := s.queries.DeleteSubnetsByIDs(ctx, generated.DeleteSubnetsByIDsParams{
		Ids:       ids,
		NetworkID: networkID,
	})
	if err != nil {
		return 0, fmt.Errorf("delete subnets by ids: %w", err)
	}
	return int64(len(deleted)), nil
}

func (s *pgSubnetStore) List(ctx context.Context, networkID int64, q db.ListQuery) (*db.ListResult[network.DBSubnet], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountSubnets(ctx, generated.CountSubnetsParams{
		NetworkID: networkID,
		Status:    filterStr(q.Filters, "status"),
		Name:      filterStr(q.Filters, "name"),
		Search:    filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count subnets: %w", err)
	}

	rows, err := s.queries.ListSubnets(ctx, generated.ListSubnetsParams{
		NetworkID:  networkID,
		Status:     filterStr(q.Filters, "status"),
		Name:       filterStr(q.Filters, "name"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list subnets: %w", err)
	}

	return &db.ListResult[network.DBSubnet]{Items: rows, TotalCount: count}, nil
}

func (s *pgSubnetStore) ListCIDRsByNetworkID(ctx context.Context, networkID int64) ([]network.DBSubnetCIDR, error) {
	rows, err := s.queries.ListSubnetCIDRsByNetworkID(ctx, networkID)
	if err != nil {
		return nil, fmt.Errorf("list subnet cidrs: %w", err)
	}
	return rows, nil
}

func (s *pgSubnetStore) CountNonGatewayAllocations(ctx context.Context, subnetID int64) (int64, error) {
	count, err := s.queries.CountNonGatewayAllocationsBySubnetID(ctx, subnetID)
	if err != nil {
		return 0, fmt.Errorf("count non-gateway allocations: %w", err)
	}
	return count, nil
}
