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

type pgWorkspaceStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGWorkspaceStore creates a new PostgreSQL-backed WorkspaceStore.
func NewPGWorkspaceStore(pool *pgxpool.Pool, queries *generated.Queries) iam.WorkspaceStore {
	return &pgWorkspaceStore{db: pool, queries: queries}
}

func (s *pgWorkspaceStore) Create(ctx context.Context, ws *iam.DBWorkspace) (*iam.DBWorkspace, error) {
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

	// Add owner to workspace as "owner"
	_, err = qtx.AddUserToWorkspace(ctx, generated.AddUserToWorkspaceParams{
		UserID:      ws.OwnerID,
		WorkspaceID: row.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to workspace: %w", err)
	}

	// Add owner to default namespace as "owner"
	_, err = qtx.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      ws.OwnerID,
		NamespaceID: defaultNS.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to default namespace: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &row, nil
}

func (s *pgWorkspaceStore) GetByID(ctx context.Context, id int64) (*iam.DBWorkspace, error) {
	row, err := s.queries.GetWorkspaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get workspace by id: %w", err)
	}
	return &row, nil
}

func (s *pgWorkspaceStore) GetByName(ctx context.Context, name string) (*iam.DBWorkspace, error) {
	row, err := s.queries.GetWorkspaceByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("workspace", name)
		}
		return nil, fmt.Errorf("get workspace by name: %w", err)
	}
	return &row, nil
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
	return &row, nil
}

func (s *pgWorkspaceStore) Delete(ctx context.Context, id int64) error {
	// Check for child namespaces
	count, err := s.queries.CountNamespacesByWorkspaceID(ctx, id)
	if err != nil {
		return fmt.Errorf("count namespaces: %w", err)
	}
	if count > 0 {
		return apierrors.NewBadRequest(
			fmt.Sprintf("cannot delete workspace %d: has %d namespace(s)", id, count),
			nil,
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

	countParams := generated.CountWorkspacesParams{
		Status:  filterStr("status"),
		Name:    filterStr("name"),
		OwnerID: filterInt64("owner_id"),
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
			OwnerUsername: r.OwnerUsername,
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
