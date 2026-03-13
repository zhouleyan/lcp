package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/o11y"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgEndpointStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGEndpointStore creates a new PostgreSQL-backed EndpointStore.
func NewPGEndpointStore(pool *pgxpool.Pool, queries *generated.Queries) o11y.EndpointStore {
	return &pgEndpointStore{pool: pool, queries: queries}
}

func (s *pgEndpointStore) Create(ctx context.Context, ep *o11y.DBEndpoint) (*o11y.DBEndpoint, error) {
	row, err := s.queries.CreateEndpoint(ctx, generated.CreateEndpointParams{
		Name:        ep.Name,
		Description: ep.Description,
		Public:      ep.Public,
		MetricsUrl:  ep.MetricsUrl,
		LogsUrl:     ep.LogsUrl,
		TracesUrl:   ep.TracesUrl,
		ApmUrl:      ep.ApmUrl,
		Status:      ep.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("endpoint", ep.Name)
		}
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	return &row, nil
}

func (s *pgEndpointStore) GetByID(ctx context.Context, id int64) (*o11y.DBEndpoint, error) {
	row, err := s.queries.GetEndpointByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("endpoint", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get endpoint by id: %w", err)
	}
	return &row, nil
}

func (s *pgEndpointStore) Update(ctx context.Context, ep *o11y.DBEndpoint) (*o11y.DBEndpoint, error) {
	row, err := s.queries.UpdateEndpoint(ctx, generated.UpdateEndpointParams{
		ID:          ep.ID,
		Name:        ep.Name,
		Description: ep.Description,
		Public:      ep.Public,
		MetricsUrl:  ep.MetricsUrl,
		LogsUrl:     ep.LogsUrl,
		TracesUrl:   ep.TracesUrl,
		ApmUrl:      ep.ApmUrl,
		Status:      ep.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("endpoint", fmt.Sprintf("%d", ep.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("endpoint", ep.Name)
		}
		return nil, fmt.Errorf("update endpoint: %w", err)
	}
	return &row, nil
}

func (s *pgEndpointStore) Patch(ctx context.Context, id int64, fields map[string]any) (*o11y.DBEndpoint, error) {
	params := generated.PatchEndpointParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["public"].(bool); ok {
		params.Public = &v
	}
	if v, ok := fields["metricsUrl"].(string); ok {
		params.MetricsUrl = &v
	}
	if v, ok := fields["logsUrl"].(string); ok {
		params.LogsUrl = &v
	}
	if v, ok := fields["tracesUrl"].(string); ok {
		params.TracesUrl = &v
	}
	if v, ok := fields["apmUrl"].(string); ok {
		params.ApmUrl = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}

	row, err := s.queries.PatchEndpoint(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("endpoint", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("endpoint", n)
			}
			return nil, apierrors.NewConflict("endpoint", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch endpoint: %w", err)
	}
	return &row, nil
}

func (s *pgEndpointStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteEndpoint(ctx, id); err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}

func (s *pgEndpointStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	count, err := s.queries.DeleteEndpointsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete endpoints by ids: %w", err)
	}
	return count, nil
}

func (s *pgEndpointStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[o11y.DBEndpoint], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountEndpoints(ctx, generated.CountEndpointsParams{
		Status: filterStr(q.Filters, "status"),
		Search: filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count endpoints: %w", err)
	}

	rows, err := s.queries.ListEndpoints(ctx, generated.ListEndpointsParams{
		Status:     filterStr(q.Filters, "status"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}

	return &db.ListResult[o11y.DBEndpoint]{Items: rows, TotalCount: count}, nil
}
