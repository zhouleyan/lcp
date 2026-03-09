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

type pgWorkspaceStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGWorkspaceStore creates a new PostgreSQL-backed WorkspaceStore.
func NewPGWorkspaceStore(pool *pgxpool.Pool, queries *generated.Queries) iam.WorkspaceStore {
	return &pgWorkspaceStore{db: pool, queries: queries}
}

func (s *pgWorkspaceStore) Create(ctx context.Context, ws *iam.DBWorkspace) (*iam.DBWorkspaceWithOwner, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	row, err := qtx.CreateWorkspace(ctx, generated.CreateWorkspaceParams{
		Name:        ws.Name,
		DisplayName: ws.DisplayName,
		Description: ws.Description,
		OwnerID:     ws.OwnerID,
		Status:      ws.Status,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("workspace", ws.Name)
		}
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Create default namespace for this workspace
	defaultNS, err := qtx.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        row.Name + "-default",
		DisplayName: "Default",
		Description: fmt.Sprintf("Default namespace for workspace %s", row.Name),
		WorkspaceID: row.ID,
		OwnerID:     ws.OwnerID,
		Visibility:  "private",
		MaxMembers:  0,
		Status:      "active",
	})
	if err != nil {
		return nil, fmt.Errorf("create default namespace: %w", err)
	}

	// Create workspace-admin role binding with is_owner=true
	wsAdminRole, err := qtx.GetRoleByName(ctx, iam.RoleWorkspaceAdmin)
	if err != nil {
		return nil, fmt.Errorf("get workspace-admin role: %w", err)
	}
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      ws.OwnerID,
		RoleID:      wsAdminRole.ID,
		Scope:       "workspace",
		WorkspaceID: &row.ID,
		IsOwner:     true,
	}); err != nil {
		return nil, fmt.Errorf("create workspace owner role binding: %w", err)
	}

	// Create namespace-admin role binding with is_owner=true for default namespace
	nsAdminRole, err := qtx.GetRoleByName(ctx, iam.RoleNamespaceAdmin)
	if err != nil {
		return nil, fmt.Errorf("get namespace-admin role: %w", err)
	}
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      ws.OwnerID,
		RoleID:      nsAdminRole.ID,
		Scope:       "namespace",
		WorkspaceID: &row.ID,
		NamespaceID: &defaultNS.ID,
		IsOwner:     true,
	}); err != nil {
		return nil, fmt.Errorf("create default namespace owner role binding: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Fetch the full workspace with owner info after commit
	wsRow, err := s.queries.GetWorkspaceByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get workspace after create: %w", err)
	}
	return &iam.DBWorkspaceWithOwner{
		Workspace: generated.Workspace{
			ID:          wsRow.ID,
			Name:        wsRow.Name,
			DisplayName: wsRow.DisplayName,
			Description: wsRow.Description,
			OwnerID:     wsRow.OwnerID,
			Status:      wsRow.Status,
			CreatedAt:   wsRow.CreatedAt,
			UpdatedAt:   wsRow.UpdatedAt,
		},
		OwnerUsername:  wsRow.OwnerUsername,
		NamespaceCount: wsRow.NamespaceCount,
		MemberCount:    wsRow.MemberCount,
	}, nil
}

func (s *pgWorkspaceStore) GetByID(ctx context.Context, id int64) (*iam.DBWorkspaceWithOwner, error) {
	row, err := s.queries.GetWorkspaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get workspace by id: %w", err)
	}
	return &iam.DBWorkspaceWithOwner{
		Workspace: generated.Workspace{
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: row.DisplayName,
			Description: row.Description,
			OwnerID:     row.OwnerID,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
		OwnerUsername:  row.OwnerUsername,
		NamespaceCount: row.NamespaceCount,
		MemberCount:    row.MemberCount,
	}, nil
}

func (s *pgWorkspaceStore) GetByName(ctx context.Context, name string) (*iam.DBWorkspace, error) {
	row, err := s.queries.GetWorkspaceByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", name)
		}
		return nil, fmt.Errorf("get workspace by name: %w", err)
	}
	return new(row), nil
}

func (s *pgWorkspaceStore) Update(ctx context.Context, ws *iam.DBWorkspace) (*iam.DBWorkspace, error) {
	row, err := s.queries.UpdateWorkspace(ctx, generated.UpdateWorkspaceParams{
		ID:          ws.ID,
		Name:        ws.Name,
		DisplayName: ws.DisplayName,
		Description: ws.Description,
		OwnerID:     ws.OwnerID,
		Status:      ws.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", fmt.Sprintf("%d", ws.ID))
		}
		return nil, fmt.Errorf("update workspace: %w", err)
	}
	return new(row), nil
}

func (s *pgWorkspaceStore) Patch(ctx context.Context, id int64, ws *iam.DBWorkspace) (*iam.DBWorkspace, error) {
	row, err := s.queries.PatchWorkspace(ctx, generated.PatchWorkspaceParams{
		ID:          id,
		Name:        toNullString(ws.Name),
		DisplayName: toNullString(ws.DisplayName),
		Description: toNullString(ws.Description),
		OwnerID:     toNullInt64(ws.OwnerID),
		Status:      toNullString(ws.Status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("patch workspace: %w", err)
	}
	return new(row), nil
}

func (s *pgWorkspaceStore) Delete(ctx context.Context, id int64) error {
	// Check for child namespaces
	count, err := s.queries.CountNamespacesByWorkspaceID(ctx, id)
	if err != nil {
		return fmt.Errorf("count namespaces: %w", err)
	}
	if count > 0 {
		return apierrors.NewConflictMessage(
			fmt.Sprintf("cannot delete workspace %d: has %d namespace(s)", id, count),
		)
	}

	if err := s.queries.DeleteWorkspace(ctx, id); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return nil
}

func (s *pgWorkspaceStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.queries.DeleteWorkspacesByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete workspaces by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgWorkspaceStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBWorkspaceWithOwner], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountWorkspacesParams{
		Status:  filterStr(q.Filters, "status"),
		Name:    filterStr(q.Filters, "name"),
		OwnerID: filterInt64(q.Filters, "owner_id"),
		Search:  filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountWorkspaces(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count workspaces: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListWorkspaces(ctx, generated.ListWorkspacesParams{
		Status:     countParams.Status,
		Name:       countParams.Name,
		OwnerID:    countParams.OwnerID,
		Search:     countParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	items := make([]iam.DBWorkspaceWithOwner, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBWorkspaceWithOwner{
			Workspace: generated.Workspace{
				ID:          r.ID,
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
				OwnerID:     r.OwnerID,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			OwnerUsername:  r.OwnerUsername,
			NamespaceCount: r.NamespaceCount,
			MemberCount:    r.MemberCount,
		})
	}

	return &db.ListResult[iam.DBWorkspaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgWorkspaceStore) CountNamespaces(ctx context.Context, workspaceID int64) (int64, error) {
	return s.queries.CountNamespacesByWorkspaceID(ctx, workspaceID)
}
