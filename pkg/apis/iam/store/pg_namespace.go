package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// ===== pgNamespaceStore =====

type pgNamespaceStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGNamespaceStore creates a new PostgreSQL-backed NamespaceStore.
func NewPGNamespaceStore(pool *pgxpool.Pool, queries *generated.Queries) iam.NamespaceStore {
	return &pgNamespaceStore{db: pool, queries: queries}
}

func (s *pgNamespaceStore) Create(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	row, err := qtx.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        ns.Name,
		DisplayName: ns.DisplayName,
		Description: ns.Description,
		OwnerID:     ns.OwnerID,
		Visibility:  ns.Visibility,
		MaxMembers:  ns.MaxMembers,
		Status:      ns.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// Auto-add owner as member with role "owner"
	_, err = qtx.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      ns.OwnerID,
		NamespaceID: row.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to namespace: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &row, nil
}

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*iam.DBNamespace, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*iam.DBNamespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", name)
		}
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	row, err := s.queries.UpdateNamespace(ctx, generated.UpdateNamespaceParams{
		ID:          ns.ID,
		Name:        ns.Name,
		DisplayName: ns.DisplayName,
		Description: ns.Description,
		OwnerID:     ns.OwnerID,
		Visibility:  ns.Visibility,
		MaxMembers:  ns.MaxMembers,
		Status:      ns.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", fmt.Sprintf("%d", ns.ID))
		}
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteNamespace(ctx, id); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (s *pgNamespaceStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteNamespacesByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete namespaces by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgNamespaceStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBNamespaceWithOwner], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	filterStr := func(key string) *string {
		if v, ok := q.Filters[key]; ok {
			if s, ok := v.(string); ok {
				return &s
			}
		}
		return nil
	}

	filterInt64 := func(key string) *int64 {
		if v, ok := q.Filters[key]; ok {
			switch val := v.(type) {
			case int64:
				return &val
			case string:
				if i, err := strconv.ParseInt(val, 10, 64); err == nil {
					return &i
				}
			}
		}
		return nil
	}

	countParams := generated.CountNamespacesParams{
		Status:     filterStr("status"),
		Name:       filterStr("name"),
		Visibility: filterStr("visibility"),
		OwnerID:    filterInt64("owner_id"),
	}

	count, err := s.queries.CountNamespaces(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count namespaces: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListNamespaces(ctx, generated.ListNamespacesParams{
		Status:     countParams.Status,
		Name:       countParams.Name,
		Visibility: countParams.Visibility,
		OwnerID:    countParams.OwnerID,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	items := make([]iam.DBNamespaceWithOwner, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBNamespaceWithOwner{
			Namespace: generated.Namespace{
				ID:          r.ID,
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
				OwnerID:     r.OwnerID,
				Visibility:  r.Visibility,
				MaxMembers:  r.MaxMembers,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			OwnerUsername: r.OwnerUsername,
		})
	}

	return &db.ListResult[iam.DBNamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

// ===== pgUserNamespaceStore =====

type pgUserNamespaceStore struct {
	queries *generated.Queries
}

// NewPGUserNamespaceStore creates a new PostgreSQL-backed UserNamespaceStore.
func NewPGUserNamespaceStore(queries *generated.Queries) iam.UserNamespaceStore {
	return &pgUserNamespaceStore{queries: queries}
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, rel *iam.DBUserNamespace) (*iam.DBUserNamespace, error) {
	row, err := s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      rel.UserID,
		NamespaceID: rel.NamespaceID,
		Role:        rel.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add user to namespace: %w", err)
	}
	return &row, nil
}

func (s *pgUserNamespaceStore) Remove(ctx context.Context, userID, namespaceID int64) error {
	if err := s.queries.RemoveUserFromNamespace(ctx, generated.RemoveUserFromNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	}); err != nil {
		return fmt.Errorf("remove user from namespace: %w", err)
	}
	return nil
}

func (s *pgUserNamespaceStore) UpdateRole(ctx context.Context, rel *iam.DBUserNamespace) (*iam.DBUserNamespace, error) {
	row, err := s.queries.UpdateUserNamespaceRole(ctx, generated.UpdateUserNamespaceRoleParams{
		UserID:      rel.UserID,
		NamespaceID: rel.NamespaceID,
		Role:        rel.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("update user namespace role: %w", err)
	}
	return &row, nil
}

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*iam.DBUserNamespace, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return &row, nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]iam.DBNamespaceWithRole, error) {
	rows, err := s.queries.ListNamespacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]iam.DBNamespaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBNamespaceWithRole{
			Namespace: generated.Namespace{
				ID:          row.ID,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Description: row.Description,
				OwnerID:     row.OwnerID,
				Visibility:  row.Visibility,
				MaxMembers:  row.MaxMembers,
				Status:      row.Status,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
			Role:     row.Role,
			JoinedAt: row.JoinedAt,
		})
	}
	return items, nil
}

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]iam.DBUserWithRole, error) {
	rows, err := s.queries.ListUsersByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("list users by namespace: %w", err)
	}

	items := make([]iam.DBUserWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBUserWithRole{
			User: generated.User{
				ID:          row.ID,
				Username:    row.Username,
				Email:       row.Email,
				DisplayName: row.DisplayName,
				Phone:       row.Phone,
				AvatarUrl:   row.AvatarUrl,
				Status:      row.Status,
				LastLoginAt: row.LastLoginAt,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
			Role:     row.Role,
			JoinedAt: row.JoinedAt,
		})
	}
	return items, nil
}
