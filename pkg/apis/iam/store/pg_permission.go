package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgPermissionStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGPermissionStore creates a new PostgreSQL-backed PermissionStore.
func NewPGPermissionStore(pool *pgxpool.Pool, queries *generated.Queries) iam.PermissionStore {
	return &pgPermissionStore{db: pool, queries: queries}
}

func (s *pgPermissionStore) Upsert(ctx context.Context, perm *iam.DBPermission) (*iam.DBPermission, error) {
	row, err := s.queries.UpsertPermission(ctx, generated.UpsertPermissionParams{
		Code:        perm.Code,
		Method:      perm.Method,
		Path:        perm.Path,
		Description: perm.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert permission: %w", err)
	}
	return &row, nil
}

func (s *pgPermissionStore) DeleteByModuleNotInCodes(ctx context.Context, modulePrefix string, keepCodes []string) error {
	if err := s.queries.DeletePermissionsByModulePrefix(ctx, generated.DeletePermissionsByModulePrefixParams{
		ModulePrefix: modulePrefix,
		KeepCodes:    keepCodes,
	}); err != nil {
		return fmt.Errorf("delete permissions by module prefix: %w", err)
	}
	return nil
}

func (s *pgPermissionStore) GetByCode(ctx context.Context, code string) (*iam.DBPermission, error) {
	row, err := s.queries.GetPermissionByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("permission", code)
		}
		return nil, fmt.Errorf("get permission by code: %w", err)
	}
	return &row, nil
}

func (s *pgPermissionStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBPermission], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountPermissionsParams{
		ModulePrefix: filterStr(q.Filters, "module_prefix"),
		Search:       filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountPermissions(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count permissions: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "asc"
	}

	rows, err := s.queries.ListPermissions(ctx, generated.ListPermissionsParams{
		ModulePrefix: countParams.ModulePrefix,
		Search:       countParams.Search,
		SortField:    q.SortBy,
		SortOrder:    sortOrder,
		PageOffset:   offset,
		PageSize:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	items := make([]iam.DBPermission, len(rows))
	for i, r := range rows {
		items[i] = r
	}

	return &db.ListResult[iam.DBPermission]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgPermissionStore) ListAllCodes(ctx context.Context) ([]string, error) {
	codes, err := s.queries.ListAllPermissionCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all permission codes: %w", err)
	}
	return codes, nil
}

func (s *pgPermissionStore) SyncModule(ctx context.Context, modulePrefix string, perms []iam.DBPermission) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		if _, err := qtx.UpsertPermission(ctx, generated.UpsertPermissionParams{
			Code:        p.Code,
			Method:      p.Method,
			Path:        p.Path,
			Description: p.Description,
		}); err != nil {
			return fmt.Errorf("upsert permission %s: %w", p.Code, err)
		}
		codes = append(codes, p.Code)
	}

	if err := qtx.DeletePermissionsByModulePrefix(ctx, generated.DeletePermissionsByModulePrefixParams{
		ModulePrefix: modulePrefix,
		KeepCodes:    codes,
	}); err != nil {
		return fmt.Errorf("cleanup stale permissions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
