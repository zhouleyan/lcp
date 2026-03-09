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

// ===== pgNamespaceStore =====

type pgNamespaceStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGNamespaceStore creates a new PostgreSQL-backed NamespaceStore.
func NewPGNamespaceStore(pool *pgxpool.Pool, queries *generated.Queries) iam.NamespaceStore {
	return &pgNamespaceStore{db: pool, queries: queries}
}

func (s *pgNamespaceStore) Create(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespaceWithOwner, error) {
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
		WorkspaceID: ns.WorkspaceID,
		OwnerID:     ns.OwnerID,
		Visibility:  ns.Visibility,
		MaxMembers:  ns.MaxMembers,
		Status:      ns.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("namespace", ns.Name)
		}
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

	// Create namespace-admin role binding with is_owner=true
	nsAdminRole, err := qtx.GetRoleByName(ctx, "namespace-admin")
	if err != nil {
		return nil, fmt.Errorf("get namespace-admin role: %w", err)
	}
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      ns.OwnerID,
		RoleID:      nsAdminRole.ID,
		Scope:       "namespace",
		WorkspaceID: &ns.WorkspaceID,
		NamespaceID: &row.ID,
		IsOwner:     true,
	}); err != nil {
		return nil, fmt.Errorf("create namespace owner role binding: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Fetch the full namespace with owner info after commit
	nsRow, err := s.queries.GetNamespaceByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get namespace after create: %w", err)
	}
	return &iam.DBNamespaceWithOwner{
		Namespace: generated.Namespace{
			ID:          nsRow.ID,
			Name:        nsRow.Name,
			DisplayName: nsRow.DisplayName,
			Description: nsRow.Description,
			WorkspaceID: nsRow.WorkspaceID,
			OwnerID:     nsRow.OwnerID,
			Visibility:  nsRow.Visibility,
			MaxMembers:  nsRow.MaxMembers,
			Status:      nsRow.Status,
			CreatedAt:   nsRow.CreatedAt,
			UpdatedAt:   nsRow.UpdatedAt,
		},
		OwnerUsername:  nsRow.OwnerUsername,
		WorkspaceName: nsRow.WorkspaceName,
		MemberCount:   nsRow.MemberCount,
	}, nil
}

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*iam.DBNamespaceWithOwner, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return &iam.DBNamespaceWithOwner{
		Namespace: generated.Namespace{
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: row.DisplayName,
			Description: row.Description,
			WorkspaceID: row.WorkspaceID,
			OwnerID:     row.OwnerID,
			Visibility:  row.Visibility,
			MaxMembers:  row.MaxMembers,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
		OwnerUsername: row.OwnerUsername,
		WorkspaceName: row.WorkspaceName,
		MemberCount:  row.MemberCount,
	}, nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*iam.DBNamespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", name)
		}
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return new(row), nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	row, err := s.queries.UpdateNamespace(ctx, generated.UpdateNamespaceParams{
		ID:          ns.ID,
		Name:        ns.Name,
		DisplayName: ns.DisplayName,
		Description: ns.Description,
		WorkspaceID: ns.WorkspaceID,
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
	return new(row), nil
}

func (s *pgNamespaceStore) Patch(ctx context.Context, id int64, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	row, err := s.queries.PatchNamespace(ctx, generated.PatchNamespaceParams{
		ID:          id,
		Name:        toNullString(ns.Name),
		DisplayName: toNullString(ns.DisplayName),
		Description: toNullString(ns.Description),
		WorkspaceID: toNullInt64(ns.WorkspaceID),
		OwnerID:     toNullInt64(ns.OwnerID),
		Visibility:  toNullString(ns.Visibility),
		MaxMembers:  toNullInt32(ns.MaxMembers),
		Status:      toNullString(ns.Status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("namespace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch namespace: %w", err)
	}
	return new(row), nil
}

func (s *pgNamespaceStore) Delete(ctx context.Context, id int64) error {
	// Check for member users
	count, err := s.queries.CountUsersByNamespaceID(ctx, id)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return apierrors.NewConflictMessage(
			fmt.Sprintf("cannot delete namespace %d: has %d member(s)", id, count),
		)
	}

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

	countParams := generated.CountNamespacesParams{
		Status:      filterStr(q.Filters, "status"),
		Name:        filterStr(q.Filters, "name"),
		Visibility:  filterStr(q.Filters, "visibility"),
		OwnerID:     filterInt64(q.Filters, "owner_id"),
		WorkspaceID: filterInt64(q.Filters, "workspace_id"),
		Search:      filterStr(q.Filters, "search"),
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
		Status:      countParams.Status,
		Name:        countParams.Name,
		Visibility:  countParams.Visibility,
		OwnerID:     countParams.OwnerID,
		WorkspaceID: countParams.WorkspaceID,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
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
				WorkspaceID: r.WorkspaceID,
				OwnerID:     r.OwnerID,
				Visibility:  r.Visibility,
				MaxMembers:  r.MaxMembers,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			OwnerUsername: r.OwnerUsername,
			WorkspaceName: r.WorkspaceName,
			MemberCount:  r.MemberCount,
		})
	}

	return &db.ListResult[iam.DBNamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgNamespaceStore) CountUsers(ctx context.Context, namespaceID int64) (int64, error) {
	return s.queries.CountUsersByNamespaceID(ctx, namespaceID)
}

// ===== pgUserNamespaceStore =====

type pgUserNamespaceStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGUserNamespaceStore creates a new PostgreSQL-backed UserNamespaceStore.
func NewPGUserNamespaceStore(pool *pgxpool.Pool, queries *generated.Queries) iam.UserNamespaceStore {
	return &pgUserNamespaceStore{db: pool, queries: queries}
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, rel *iam.DBUserNamespace) (*iam.DBUserNamespace, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Get namespace to find workspace_id
	ns, err := qtx.GetNamespaceByID(ctx, rel.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	// Auto-add user to workspace (ON CONFLICT DO NOTHING handles duplicates)
	_, err = qtx.AddUserToWorkspace(ctx, generated.AddUserToWorkspaceParams{
		UserID:      rel.UserID,
		WorkspaceID: ns.WorkspaceID,
		Role:        "member",
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("auto-add user to workspace: %w", err)
	}

	row, err := qtx.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      rel.UserID,
		NamespaceID: rel.NamespaceID,
		Role:        rel.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING — user already a member
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit transaction: %w", err)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("add user to namespace: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return new(row), nil
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
	return new(row), nil
}

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*iam.DBUserNamespace, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return new(row), nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64, q db.ListQuery) (*db.ListResult[iam.DBNamespaceWithOwnerAndRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountNamespacesByUserIDJoinedParams{
		UserID:      userID,
		Status:      filterStr(q.Filters, "status"),
		Visibility:  filterStr(q.Filters, "visibility"),
		WorkspaceID: filterInt64(q.Filters, "workspace_id"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountNamespacesByUserIDJoined(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count namespaces by user: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListNamespacesByUserIDPaginated(ctx, generated.ListNamespacesByUserIDPaginatedParams{
		UserID:      userID,
		Status:      countParams.Status,
		Visibility:  countParams.Visibility,
		WorkspaceID: countParams.WorkspaceID,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]iam.DBNamespaceWithOwnerAndRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBNamespaceWithOwnerAndRole{
			Namespace: generated.Namespace{
				ID:          row.ID,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Description: row.Description,
				WorkspaceID: row.WorkspaceID,
				OwnerID:     row.OwnerID,
				Visibility:  row.Visibility,
				MaxMembers:  row.MaxMembers,
				Status:      row.Status,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
			OwnerUsername:  row.OwnerUsername,
			WorkspaceName: row.WorkspaceName,
			MemberCount:   row.MemberCount,
			Role:           row.Role,
			JoinedAt:       row.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBNamespaceWithOwnerAndRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64, q db.ListQuery) (*db.ListResult[iam.DBUserWithRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountUsersByNamespaceIDFilteredParams{
		NamespaceID: namespaceID,
		Status:      filterStr(q.Filters, "status"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountUsersByNamespaceIDFiltered(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count users by namespace: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListUsersByNamespaceIDPaginated(ctx, generated.ListUsersByNamespaceIDPaginatedParams{
		NamespaceID: namespaceID,
		Status:      countParams.Status,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
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

	return &db.ListResult[iam.DBUserWithRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}
