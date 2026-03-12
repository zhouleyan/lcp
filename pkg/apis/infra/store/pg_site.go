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

type pgSiteStore struct {
	queries *generated.Queries
}

// NewPGSiteStore creates a new PostgreSQL-backed SiteStore.
func NewPGSiteStore(queries *generated.Queries) infra.SiteStore {
	return &pgSiteStore{queries: queries}
}

func (s *pgSiteStore) Create(ctx context.Context, site *infra.DBSite) (*infra.DBSite, error) {
	row, err := s.queries.CreateSite(ctx, generated.CreateSiteParams{
		Name:         site.Name,
		DisplayName:  site.DisplayName,
		Description:  site.Description,
		RegionID:     site.RegionID,
		Status:       site.Status,
		Address:      site.Address,
		Latitude:     site.Latitude,
		Longitude:    site.Longitude,
		ContactName:  site.ContactName,
		ContactPhone: site.ContactPhone,
		ContactEmail: site.ContactEmail,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("site", site.Name)
		}
		return nil, fmt.Errorf("create site: %w", err)
	}
	return &row, nil
}

func (s *pgSiteStore) GetByID(ctx context.Context, id int64) (*infra.DBSiteWithDetails, error) {
	row, err := s.queries.GetSiteByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("site", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get site by id: %w", err)
	}
	return &row, nil
}

func (s *pgSiteStore) Update(ctx context.Context, site *infra.DBSite) (*infra.DBSite, error) {
	row, err := s.queries.UpdateSite(ctx, generated.UpdateSiteParams{
		ID:           site.ID,
		Name:         site.Name,
		DisplayName:  site.DisplayName,
		Description:  site.Description,
		RegionID:     site.RegionID,
		Status:       site.Status,
		Address:      site.Address,
		Latitude:     site.Latitude,
		Longitude:    site.Longitude,
		ContactName:  site.ContactName,
		ContactPhone: site.ContactPhone,
		ContactEmail: site.ContactEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("site", fmt.Sprintf("%d", site.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("site", site.Name)
		}
		return nil, fmt.Errorf("update site: %w", err)
	}
	return &row, nil
}

func (s *pgSiteStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBSite, error) {
	params := generated.PatchSiteParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["regionId"].(int64); ok {
		params.RegionID = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}
	if v, ok := fields["address"].(string); ok {
		params.Address = &v
	}
	if v, ok := fields["latitude"].(*float64); ok {
		params.Latitude = v
	}
	if v, ok := fields["longitude"].(*float64); ok {
		params.Longitude = v
	}
	if v, ok := fields["contactName"].(string); ok {
		params.ContactName = &v
	}
	if v, ok := fields["contactPhone"].(string); ok {
		params.ContactPhone = &v
	}
	if v, ok := fields["contactEmail"].(string); ok {
		params.ContactEmail = &v
	}

	row, err := s.queries.PatchSite(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("site", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("site", n)
			}
			return nil, apierrors.NewConflict("site", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch site: %w", err)
	}
	return &row, nil
}

func (s *pgSiteStore) Delete(ctx context.Context, id int64) error {
	count, err := s.queries.CountSiteChildLocations(ctx, id)
	if err != nil {
		return fmt.Errorf("count site child locations: %w", err)
	}
	if count > 0 {
		return apierrors.NewBadRequest(fmt.Sprintf("cannot delete site: has %d child locations", count), nil)
	}

	if err := s.queries.DeleteSite(ctx, id); err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	return nil
}

func (s *pgSiteStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		count, err := s.queries.CountSiteChildLocations(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("count site child locations for id %d: %w", id, err)
		}
		if count > 0 {
			return 0, apierrors.NewBadRequest(fmt.Sprintf("cannot delete site %d: has %d child locations", id, count), nil)
		}
	}

	deletedIDs, err := s.queries.DeleteSitesByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete sites by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgSiteStore) CountChildLocations(ctx context.Context, siteID int64) (int64, error) {
	count, err := s.queries.CountSiteChildLocations(ctx, siteID)
	if err != nil {
		return 0, fmt.Errorf("count site child locations: %w", err)
	}
	return count, nil
}

func (s *pgSiteStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBSiteListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountSites(ctx, generated.CountSitesParams{
		RegionID: filterInt64(q.Filters, "regionId"),
		Status:   filterStr(q.Filters, "status"),
		Search:   filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count sites: %w", err)
	}

	rows, err := s.queries.ListSites(ctx, generated.ListSitesParams{
		RegionID:   filterInt64(q.Filters, "regionId"),
		Status:     filterStr(q.Filters, "status"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	return &db.ListResult[infra.DBSiteListRow]{Items: rows, TotalCount: count}, nil
}

