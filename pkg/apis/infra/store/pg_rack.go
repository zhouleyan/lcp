package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgRackStore struct {
	queries *generated.Queries
}

// NewPGRackStore creates a new PostgreSQL-backed RackStore.
func NewPGRackStore(queries *generated.Queries) infra.RackStore {
	return &pgRackStore{queries: queries}
}

func (s *pgRackStore) Create(ctx context.Context, rack *infra.DBRack) (*infra.DBRack, error) {
	row, err := s.queries.CreateRack(ctx, generated.CreateRackParams{
		Name:          rack.Name,
		DisplayName:   rack.DisplayName,
		Description:   rack.Description,
		LocationID:    rack.LocationID,
		Status:        rack.Status,
		UHeight:       rack.UHeight,
		Position:      rack.Position,
		PowerCapacity: rack.PowerCapacity,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("rack", rack.Name)
		}
		return nil, fmt.Errorf("create rack: %w", err)
	}
	return &row, nil
}

func (s *pgRackStore) GetByID(ctx context.Context, id int64) (*infra.DBRackWithDetails, error) {
	row, err := s.queries.GetRackByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("rack", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get rack by id: %w", err)
	}
	return &row, nil
}

func (s *pgRackStore) Update(ctx context.Context, rack *infra.DBRack) (*infra.DBRack, error) {
	row, err := s.queries.UpdateRack(ctx, generated.UpdateRackParams{
		ID:            rack.ID,
		Name:          rack.Name,
		DisplayName:   rack.DisplayName,
		Description:   rack.Description,
		LocationID:    rack.LocationID,
		Status:        rack.Status,
		UHeight:       rack.UHeight,
		Position:      rack.Position,
		PowerCapacity: rack.PowerCapacity,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("rack", fmt.Sprintf("%d", rack.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("rack", rack.Name)
		}
		return nil, fmt.Errorf("update rack: %w", err)
	}
	return &row, nil
}

func (s *pgRackStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBRack, error) {
	params := generated.PatchRackParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["locationId"].(int64); ok {
		params.LocationID = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}
	if v, ok := fields["uHeight"].(int32); ok {
		params.UHeight = &v
	}
	if v, ok := fields["position"].(string); ok {
		params.Position = &v
	}
	if v, ok := fields["powerCapacity"].(string); ok {
		params.PowerCapacity = &v
	}

	row, err := s.queries.PatchRack(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("rack", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("rack", n)
			}
			return nil, apierrors.NewConflict("rack", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch rack: %w", err)
	}
	return &row, nil
}

func (s *pgRackStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteRack(ctx, id); err != nil {
		return fmt.Errorf("delete rack: %w", err)
	}
	return nil
}

func (s *pgRackStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteRacksByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete racks by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgRackStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBRackListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountRacks(ctx, generated.CountRacksParams{
		LocationID: filterInt64(q.Filters, "locationId"),
		SiteID:     filterInt64(q.Filters, "siteId"),
		RegionID:   filterInt64(q.Filters, "regionId"),
		Status:     filterStr(q.Filters, "status"),
		Search:     filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count racks: %w", err)
	}

	rows, err := s.queries.ListRacks(ctx, generated.ListRacksParams{
		LocationID: filterInt64(q.Filters, "locationId"),
		SiteID:     filterInt64(q.Filters, "siteId"),
		RegionID:   filterInt64(q.Filters, "regionId"),
		Status:     filterStr(q.Filters, "status"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list racks: %w", err)
	}

	return &db.ListResult[infra.DBRackListRow]{Items: rows, TotalCount: count}, nil
}
