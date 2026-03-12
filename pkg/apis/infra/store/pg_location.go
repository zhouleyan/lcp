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

type pgLocationStore struct {
	queries *generated.Queries
}

// NewPGLocationStore creates a new PostgreSQL-backed LocationStore.
func NewPGLocationStore(queries *generated.Queries) infra.LocationStore {
	return &pgLocationStore{queries: queries}
}

func (s *pgLocationStore) Create(ctx context.Context, location *infra.DBLocation) (*infra.DBLocation, error) {
	row, err := s.queries.CreateLocation(ctx, generated.CreateLocationParams{
		Name:         location.Name,
		DisplayName:  location.DisplayName,
		Description:  location.Description,
		SiteID:       location.SiteID,
		Status:       location.Status,
		Floor:        location.Floor,
		RackCapacity: location.RackCapacity,
		ContactName:  location.ContactName,
		ContactPhone: location.ContactPhone,
		ContactEmail: location.ContactEmail,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("location", location.Name)
		}
		return nil, fmt.Errorf("create location: %w", err)
	}
	return &row, nil
}

func (s *pgLocationStore) GetByID(ctx context.Context, id int64) (*infra.DBLocationWithDetails, error) {
	row, err := s.queries.GetLocationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("location", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get location by id: %w", err)
	}
	return &row, nil
}

func (s *pgLocationStore) Update(ctx context.Context, location *infra.DBLocation) (*infra.DBLocation, error) {
	row, err := s.queries.UpdateLocation(ctx, generated.UpdateLocationParams{
		ID:           location.ID,
		Name:         location.Name,
		DisplayName:  location.DisplayName,
		Description:  location.Description,
		SiteID:       location.SiteID,
		Status:       location.Status,
		Floor:        location.Floor,
		RackCapacity: location.RackCapacity,
		ContactName:  location.ContactName,
		ContactPhone: location.ContactPhone,
		ContactEmail: location.ContactEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("location", fmt.Sprintf("%d", location.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("location", location.Name)
		}
		return nil, fmt.Errorf("update location: %w", err)
	}
	return &row, nil
}

func (s *pgLocationStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBLocation, error) {
	params := generated.PatchLocationParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["siteId"].(int64); ok {
		params.SiteID = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}
	if v, ok := fields["floor"].(string); ok {
		params.Floor = &v
	}
	if v, ok := fields["rackCapacity"].(int32); ok {
		params.RackCapacity = &v
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

	row, err := s.queries.PatchLocation(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("location", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("location", n)
			}
			return nil, apierrors.NewConflict("location", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch location: %w", err)
	}
	return &row, nil
}

func (s *pgLocationStore) Delete(ctx context.Context, id int64) error {
	count, err := s.queries.CountLocationChildRacks(ctx, id)
	if err != nil {
		return fmt.Errorf("count location child racks: %w", err)
	}
	if count > 0 {
		return apierrors.NewBadRequest(fmt.Sprintf("cannot delete location: has %d child racks", count), nil)
	}

	if err := s.queries.DeleteLocation(ctx, id); err != nil {
		return fmt.Errorf("delete location: %w", err)
	}
	return nil
}

func (s *pgLocationStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		count, err := s.queries.CountLocationChildRacks(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("count location child racks for id %d: %w", id, err)
		}
		if count > 0 {
			return 0, apierrors.NewBadRequest(fmt.Sprintf("cannot delete location %d: has %d child racks", id, count), nil)
		}
	}

	deletedIDs, err := s.queries.DeleteLocationsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete locations by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgLocationStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBLocationListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountLocations(ctx, generated.CountLocationsParams{
		SiteID:   filterInt64(q.Filters, "siteId"),
		RegionID: filterInt64(q.Filters, "regionId"),
		Status:   filterStr(q.Filters, "status"),
		Search:   filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count locations: %w", err)
	}

	rows, err := s.queries.ListLocations(ctx, generated.ListLocationsParams{
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
		return nil, fmt.Errorf("list locations: %w", err)
	}

	return &db.ListResult[infra.DBLocationListRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgLocationStore) CountChildRacks(ctx context.Context, locationID int64) (int64, error) {
	count, err := s.queries.CountLocationChildRacks(ctx, locationID)
	if err != nil {
		return 0, fmt.Errorf("count location child racks: %w", err)
	}
	return count, nil
}

