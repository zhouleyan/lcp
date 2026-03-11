package store

import (
	"context"
	"encoding/json"
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

type pgHostStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGHostStore creates a new PostgreSQL-backed HostStore.
func NewPGHostStore(pool *pgxpool.Pool, queries *generated.Queries) infra.HostStore {
	return &pgHostStore{pool: pool, queries: queries}
}

func (s *pgHostStore) Create(ctx context.Context, host *infra.DBHost) (*infra.DBHost, error) {
	row, err := s.queries.CreateHost(ctx, generated.CreateHostParams{
		Name:        host.Name,
		DisplayName: host.DisplayName,
		Description: host.Description,
		Hostname:    host.Hostname,
		IpAddress:   host.IpAddress,
		Os:          host.Os,
		Arch:        host.Arch,
		CpuCores:    host.CpuCores,
		MemoryMb:    host.MemoryMb,
		DiskGb:      host.DiskGb,
		Labels:      host.Labels,
		Scope:       host.Scope,
		WorkspaceID: host.WorkspaceID,
		NamespaceID: host.NamespaceID,
		Status:      host.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("host", host.Name)
		}
		return nil, fmt.Errorf("create host: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) GetByID(ctx context.Context, id int64) (*infra.DBHostWithEnv, error) {
	row, err := s.queries.GetHostByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get host by id: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Update(ctx context.Context, host *infra.DBHost) (*infra.DBHost, error) {
	row, err := s.queries.UpdateHost(ctx, generated.UpdateHostParams{
		ID:          host.ID,
		Name:        host.Name,
		DisplayName: host.DisplayName,
		Description: host.Description,
		Hostname:    host.Hostname,
		IpAddress:   host.IpAddress,
		Os:          host.Os,
		Arch:        host.Arch,
		CpuCores:    host.CpuCores,
		MemoryMb:    host.MemoryMb,
		DiskGb:      host.DiskGb,
		Labels:      host.Labels,
		Status:      host.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", host.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("host", host.Name)
		}
		return nil, fmt.Errorf("update host: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Patch(ctx context.Context, id int64, fields map[string]any) (*infra.DBHost, error) {
	params := generated.PatchHostParams{ID: id}

	if v, ok := fields["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := fields["displayName"].(string); ok {
		params.DisplayName = &v
	}
	if v, ok := fields["description"].(string); ok {
		params.Description = &v
	}
	if v, ok := fields["hostname"].(string); ok {
		params.Hostname = &v
	}
	if v, ok := fields["ipAddress"].(string); ok {
		params.IpAddress = &v
	}
	if v, ok := fields["os"].(string); ok {
		params.Os = &v
	}
	if v, ok := fields["arch"].(string); ok {
		params.Arch = &v
	}
	if v, ok := fields["cpuCores"].(float64); ok {
		i := int32(v)
		params.CpuCores = &i
	}
	if v, ok := fields["memoryMb"].(float64); ok {
		i := int64(v)
		params.MemoryMb = &i
	}
	if v, ok := fields["diskGb"].(float64); ok {
		i := int64(v)
		params.DiskGb = &i
	}
	if v, ok := fields["labels"].(map[string]string); ok {
		params.Labels = labelsToJSON(v)
	} else if v, ok := fields["labels"].(map[string]any); ok {
		b, _ := json.Marshal(v)
		params.Labels = b
	}
	if v, ok := fields["status"].(string); ok {
		params.Status = &v
	}

	row, err := s.queries.PatchHost(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("host", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			if n, ok := fields["name"].(string); ok {
				return nil, apierrors.NewConflict("host", n)
			}
			return nil, apierrors.NewConflict("host", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch host: %w", err)
	}
	return &row, nil
}

func (s *pgHostStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteHost(ctx, id); err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	return nil
}

func (s *pgHostStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteHostsByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete hosts by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgHostStore) ListPlatform(ctx context.Context, q db.ListQuery) (*db.ListResult[infra.DBHostPlatformRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsPlatform(ctx, generated.CountHostsPlatformParams{
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count platform hosts: %w", err)
	}

	rows, err := s.queries.ListHostsPlatform(ctx, generated.ListHostsPlatformParams{
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list platform hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostPlatformRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) ListByWorkspaceID(ctx context.Context, wsID int64, q db.ListQuery) (*db.ListResult[infra.DBHostWorkspaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsByWorkspaceID(ctx, generated.CountHostsByWorkspaceIDParams{
		WorkspaceID:   &wsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count workspace hosts: %w", err)
	}

	rows, err := s.queries.ListHostsByWorkspaceID(ctx, generated.ListHostsByWorkspaceIDParams{
		WorkspaceID:   &wsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostWorkspaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) ListByNamespaceID(ctx context.Context, nsID int64, q db.ListQuery) (*db.ListResult[infra.DBHostNamespaceRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	count, err := s.queries.CountHostsByNamespaceID(ctx, generated.CountHostsByNamespaceIDParams{
		NamespaceID:   &nsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
	})
	if err != nil {
		return nil, fmt.Errorf("count namespace hosts: %w", err)
	}

	rows, err := s.queries.ListHostsByNamespaceID(ctx, generated.ListHostsByNamespaceIDParams{
		NamespaceID:   &nsID,
		Status:        filterStr(q.Filters, "status"),
		EnvironmentID: filterInt64(q.Filters, "environmentId"),
		Search:        filterStr(q.Filters, "search"),
		SortField:     q.SortBy,
		SortOrder:     sortOrder,
		PageOffset:    offset,
		PageSize:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespace hosts: %w", err)
	}

	return &db.ListResult[infra.DBHostNamespaceRow]{Items: rows, TotalCount: count}, nil
}

func (s *pgHostStore) BindEnvironment(ctx context.Context, hostID, envID int64) error {
	return s.queries.BindHostEnvironment(ctx, generated.BindHostEnvironmentParams{
		ID:            hostID,
		EnvironmentID: &envID,
	})
}

func (s *pgHostStore) UnbindEnvironment(ctx context.Context, hostID int64) error {
	return s.queries.UnbindHostEnvironment(ctx, hostID)
}
