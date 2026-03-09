package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgRoleStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGRoleStore creates a new PostgreSQL-backed RoleStore.
func NewPGRoleStore(pool *pgxpool.Pool, queries *generated.Queries) iam.RoleStore {
	return &pgRoleStore{db: pool, queries: queries}
}

func (s *pgRoleStore) Create(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.CreateRole(ctx, generated.CreateRoleParams{
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Scope:       role.Scope,
		Builtin:     role.Builtin,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("role", role.Name)
		}
		return nil, fmt.Errorf("create role: %w", err)
	}
	return &row, nil
}

func (s *pgRoleStore) GetByID(ctx context.Context, id int64) (*iam.DBRoleWithRules, error) {
	row, err := s.queries.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get role by id: %w", err)
	}

	rules, err := s.queries.GetRulesByRoleID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get rules for role %d: %w", id, err)
	}

	return &iam.DBRoleWithRules{
		Role:  row,
		Rules: rules,
	}, nil
}

func (s *pgRoleStore) GetByName(ctx context.Context, name string) (*iam.DBRole, error) {
	row, err := s.queries.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", name)
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return &row, nil
}

func (s *pgRoleStore) Update(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.UpdateRole(ctx, generated.UpdateRoleParams{
		ID:          role.ID,
		DisplayName: role.DisplayName,
		Description: role.Description,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", fmt.Sprintf("%d", role.ID))
		}
		return nil, fmt.Errorf("update role: %w", err)
	}
	return &row, nil
}

func (s *pgRoleStore) Upsert(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.UpsertRole(ctx, generated.UpsertRoleParams{
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Scope:       role.Scope,
		Builtin:     role.Builtin,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert role: %w", err)
	}
	return &row, nil
}

func (s *pgRoleStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (s *pgRoleStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountRolesParams{
		Scope:   filterStr(q.Filters, "scope"),
		Builtin: filterBool(q.Filters, "builtin"),
		Search:  filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoles(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count roles: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoles(ctx, generated.ListRolesParams{
		Scope:      countParams.Scope,
		Builtin:    countParams.Builtin,
		Search:     countParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	items := make([]iam.DBRole, len(rows))
	for i, r := range rows {
		items[i] = r
	}

	return &db.ListResult[iam.DBRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleStore) SetPermissionRules(ctx context.Context, roleID int64, patterns []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Delete all existing rules for this role
	if err := qtx.DeleteRolePermissionRules(ctx, roleID); err != nil {
		return fmt.Errorf("delete existing rules: %w", err)
	}

	// Insert new rules
	for _, pattern := range patterns {
		if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
			RoleID:  roleID,
			Pattern: pattern,
		}); err != nil {
			return fmt.Errorf("add rule %q: %w", pattern, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
