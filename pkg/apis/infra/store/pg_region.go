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

type pgRegionStore struct {
	queries *generated.Queries
}

// NewPGRegionStore creates a new PostgreSQL-backed RegionStore.
func NewPGRegionStore(queries *generated.Queries) infra.RegionStore {
	return &pgRegionStore{queries: queries}
}

func (s *pgRegionStore) Create(ctx context.Context, region *infra.DBRegion) (*infra.DBRegion, error) {
	row, err := s.queries.CreateRegion(ctx, generated.CreateRegionParams{
		Name:        region.Name,
		DisplayName: region.DisplayName,
		Description: region.Description,
		Status:      region.Status,
		Latitude:    region.Latitude,
		Longitude:   region.Longitude,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("region", region.Name)
		}
		return nil, fmt.Errorf("create region: %w", err)
	}
	return &row, nil
}

func (s *pgRegionStore) GetByID(ctx context.Context, id int64) (*infra.DBRegionWithCounts, error) {
	row, err := s.queries.GetRegionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("region", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get region by id: %w", err)
	}
	return &row, nil
}

func (s *pgRegionStore) Update(ctx context.Context, region *infra.DBRegion) (*infra.DBRegion, error) {
	row, err := s.queries.UpdateRegion(ctx, generated.UpdateRegionParams{
		ID:          region.ID,
		Name:        region.Name,
		DisplayName: region.DisplayName,
		Description: region.Description,
		Status:      region.Status,
		Latitude:    region.Latitude,
		Longitude:   region.Longitude,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("region", fmt.Sprintf("%d", region.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("region", region.Name)
		}
		return nil, fmt.Errorf("update region: %w", err)
	}
	return &row, nil
}

func (s *pgRegionStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBRegion, error) {
	params := generated.PatchRegionParams{ID: id}

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
	if v, ok := fields["latitude"].(*float64); ok {
		params.Latitude = v
	}
	if v, ok := fields["longitude"].(*float64); ok {
		params.Longitude = v
	}

	row, err := s.queries.PatchRegion(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("region", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("region", n)
			}
			return nil, apierrors.NewConflict("region", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch region: %w", err)
	}
	return &row, nil
}

func (s *pgRegionStore) Delete(ctx context.Context, id int64) error {
	count, err := s.queries.CountRegionChildSites(ctx, id)
	if err != nil {
		return fmt.Errorf("count region child sites: %w", err)
	}
	if count > 0 {
		return apierrors.NewBadRequest(fmt.Sprintf("cannot delete region: has %d child sites", count), nil)
	}

	if err := s.queries.DeleteRegion(ctx, id); err != nil {
		return fmt.Errorf("delete region: %w", err)
	}
	return nil
}

func (s *pgRegionStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		count, err := s.queries.CountRegionChildSites(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("count region child sites for id %d: %w", id, err)
		}
		if count > 0 {
			return 0, apierrors.NewBadRequest(fmt.Sprintf("cannot delete region %d: has %d child sites", id, count), nil)
		}
	}

	deletedIDs, err := s.queries.DeleteRegionsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete regions by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgRegionStore) CountChildSites(ctx context.Context, regionID int64) (int64, error) {
	count, err := s.queries.CountRegionChildSites(ctx, regionID)
	if err != nil {
		return 0, fmt.Errorf("count region child sites: %w", err)
	}
	return count, nil
}

func (s *pgRegionStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBRegionListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountRegions(ctx, generated.CountRegionsParams{
		Status: filterStr(q.Filters, "status"),
		Search: filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count regions: %w", err)
	}

	rows, err := s.queries.ListRegions(ctx, generated.ListRegionsParams{
		Status:     filterStr(q.Filters, "status"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list regions: %w", err)
	}

	return &db.ListResult[infra.DBRegionListRow]{Items: rows, TotalCount: count}, nil
}
