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

type pgNetworkStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGNetworkStore 创建 PostgreSQL 实现的 NetworkStore。
func NewPGNetworkStore(pool *pgxpool.Pool, queries *generated.Queries) network.NetworkStore {
	return &pgNetworkStore{pool: pool, queries: queries}
}

func (s *pgNetworkStore) Create(ctx context.Context, n *network.DBNetwork) (*network.DBNetwork, error) {
	row, err := s.queries.CreateNetwork(ctx, generated.CreateNetworkParams{
		Name:        n.Name,
		DisplayName: n.DisplayName,
		Description: n.Description,
		Cidr:        n.Cidr,
		MaxSubnets:  n.MaxSubnets,
		IsPublic:    n.IsPublic,
		Status:      n.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("network", n.Name)
		}
		return nil, fmt.Errorf("create network: %w", err)
	}
	return &row, nil
}

func (s *pgNetworkStore) GetByID(ctx context.Context, id int64) (*network.DBNetworkWithCount, error) {
	row, err := s.queries.GetNetworkByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("network", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get network by id: %w", err)
	}
	return &row, nil
}

func (s *pgNetworkStore) Update(ctx context.Context, n *network.DBNetwork) (*network.DBNetwork, error) {
	row, err := s.queries.UpdateNetwork(ctx, generated.UpdateNetworkParams{
		ID:          n.ID,
		Name:        n.Name,
		DisplayName: n.DisplayName,
		Description: n.Description,
		Status:      n.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("network", fmt.Sprintf("%d", n.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("network", n.Name)
		}
		return nil, fmt.Errorf("update network: %w", err)
	}
	return &row, nil
}

func (s *pgNetworkStore) Patch(ctx context.Context, id int64, fields map[string]any) (*network.DBNetwork, error) {
	params := generated.PatchNetworkParams{ID: id}
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

	row, err := s.queries.PatchNetwork(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("network", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("network", "")
		}
		return nil, fmt.Errorf("patch network: %w", err)
	}
	return &row, nil
}

func (s *pgNetworkStore) Delete(ctx context.Context, id int64) error {
	err := s.queries.DeleteNetwork(ctx, id)
	if err != nil {
		return fmt.Errorf("delete network: %w", err)
	}
	return nil
}

func (s *pgNetworkStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	deleted, err := s.queries.DeleteNetworksByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete networks by ids: %w", err)
	}
	return int64(len(deleted)), nil
}

func (s *pgNetworkStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[network.DBNetworkListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountNetworks(ctx, generated.CountNetworksParams{
		Status: filterStr(q.Filters, "status"),
		Name:   filterStr(q.Filters, "name"),
		Search: filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count networks: %w", err)
	}

	rows, err := s.queries.ListNetworks(ctx, generated.ListNetworksParams{
		Status:     filterStr(q.Filters, "status"),
		Name:       filterStr(q.Filters, "name"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}

	return &db.ListResult[network.DBNetworkListRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgNetworkStore) CountSubnets(ctx context.Context, networkID int64) (int64, error) {
	count, err := s.queries.CountSubnetsByNetworkID(ctx, networkID)
	if err != nil {
		return 0, fmt.Errorf("count subnets by network id: %w", err)
	}
	return count, nil
}
