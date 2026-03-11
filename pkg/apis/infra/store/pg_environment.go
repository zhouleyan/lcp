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
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgEnvironmentStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGEnvironmentStore creates a new PostgreSQL-backed EnvironmentStore.
func NewPGEnvironmentStore(pool *pgxpool.Pool, queries *generated.Queries) infra.EnvironmentStore {
	return &pgEnvironmentStore{pool: pool, queries: queries}
}

func (s *pgEnvironmentStore) Create(ctx context.Context, env *infra.DBEnvironment) (*infra.DBEnvironment, error) {
	row, err := s.queries.CreateEnvironment(ctx, generated.CreateEnvironmentParams{
		Name:        env.Name,
		DisplayName: env.DisplayName,
		Description: env.Description,
		EnvType:     env.EnvType,
		Scope:       env.Scope,
		WorkspaceID: env.WorkspaceID,
		NamespaceID: env.NamespaceID,
		Status:      env.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("environment", env.Name)
		}
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return &row, nil
}

func (s *pgEnvironmentStore) GetByID(ctx context.Context, id int64) (*infra.DBEnvWithCounts, error) {
	row, err := s.queries.GetEnvironmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("environment", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get environment by id: %w", err)
	}
	return &row, nil
}

func (s *pgEnvironmentStore) Update(ctx context.Context, env *infra.DBEnvironment) (*infra.DBEnvironment, error) {
	row, err := s.queries.UpdateEnvironment(ctx, generated.UpdateEnvironmentParams{
		ID:          env.ID,
		Name:        env.Name,
		DisplayName: env.DisplayName,
		Description: env.Description,
		EnvType:     env.EnvType,
		Status:      env.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("environment", fmt.Sprintf("%d", env.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("environment", env.Name)
		}
		return nil, fmt.Errorf("update environment: %w", err)
	}
	return &row, nil
}

func (s *pgEnvironmentStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBEnvironment, error) {
	params := generated.PatchEnvironmentParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["envType"].(string); ok {
		params.EnvType = &v
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}

	row, err := s.queries.PatchEnvironment(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("environment", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("environment", n)
			}
			return nil, apierrors.NewConflict("environment", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch environment: %w", err)
	}
	return &row, nil
}

func (s *pgEnvironmentStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteEnvironment(ctx, id); err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	return nil
}

func (s *pgEnvironmentStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteEnvironmentsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete environments by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgEnvironmentStore) ListPlatform(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBEnvPlatformRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountEnvironmentsPlatform(ctx, generated.CountEnvironmentsPlatformParams{
		Status:  filterStr(q.Filters, "status"),
		EnvType: filterStr(q.Filters, "envType"),
		Search:  filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count platform environments: %w", err)
	}

	rows, err := s.queries.ListEnvironmentsPlatform(ctx, generated.ListEnvironmentsPlatformParams{
		Status:     filterStr(q.Filters, "status"),
		EnvType:    filterStr(q.Filters, "envType"),
		Search:     filterStr(q.Filters, "search"),
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list platform environments: %w", err)
	}

	return &db.ListResult[infra.DBEnvPlatformRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgEnvironmentStore) ListByWorkspaceID(ctx context.Context, wsID int64, q db.ListQuery) (*db.ListResult[infra.DBEnvWorkspaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountEnvironmentsByWorkspaceID(ctx, generated.CountEnvironmentsByWorkspaceIDParams{
		WorkspaceID: &wsID,
		Status:      filterStr(q.Filters, "status"),
		EnvType:     filterStr(q.Filters, "envType"),
		Search:      filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count workspace environments: %w", err)
	}

	rows, err := s.queries.ListEnvironmentsByWorkspaceID(ctx, generated.ListEnvironmentsByWorkspaceIDParams{
		WorkspaceID: &wsID,
		Status:      filterStr(q.Filters, "status"),
		EnvType:     filterStr(q.Filters, "envType"),
		Search:      filterStr(q.Filters, "search"),
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace environments: %w", err)
	}

	return &db.ListResult[infra.DBEnvWorkspaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgEnvironmentStore) ListByNamespaceID(ctx context.Context, nsID int64, q db.ListQuery) (*db.ListResult[infra.DBEnvNamespaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountEnvironmentsByNamespaceID(ctx, generated.CountEnvironmentsByNamespaceIDParams{
		NamespaceID: &nsID,
		Status:      filterStr(q.Filters, "status"),
		EnvType:     filterStr(q.Filters, "envType"),
		Search:      filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count namespace environments: %w", err)
	}

	rows, err := s.queries.ListEnvironmentsByNamespaceID(ctx, generated.ListEnvironmentsByNamespaceIDParams{
		NamespaceID: &nsID,
		Status:      filterStr(q.Filters, "status"),
		EnvType:     filterStr(q.Filters, "envType"),
		Search:      filterStr(q.Filters, "search"),
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespace environments: %w", err)
	}

	return &db.ListResult[infra.DBEnvNamespaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgEnvironmentStore) ListHostsByEnvID(ctx context.Context, envID int64, q db.ListQuery) (*db.ListResult[infra.DBHostByEnvRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsByEnvironmentID(ctx, generated.CountHostsByEnvironmentIDParams{
		EnvironmentID: &envID,
		Status:        filterStr(q.Filters, "status"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count hosts by environment: %w", err)
	}

	rows, err := s.queries.ListHostsByEnvironmentID(ctx, generated.ListHostsByEnvironmentIDParams{
		EnvironmentID: &envID,
		Status:        filterStr(q.Filters, "status"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list hosts by environment: %w", err)
	}

	return &db.ListResult[infra.DBHostByEnvRow]{Items: rows, TotalCount: count}, nil
}
