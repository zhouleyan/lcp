package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgUserWorkspaceStore struct {
	queries *generated.Queries
}

// NewPGUserWorkspaceStore creates a new PostgreSQL-backed UserWorkspaceStore.
func NewPGUserWorkspaceStore(queries *generated.Queries) iam.UserWorkspaceStore {
	return &pgUserWorkspaceStore{queries: queries}
}

func (s *pgUserWorkspaceStore) Add(ctx context.Context, rel *iam.DBUserWorkspace) (*iam.DBUserWorkspace, error) {
	row, err := s.queries.AddUserToWorkspace(ctx, generated.AddUserToWorkspaceParams{
		UserID:      rel.UserID,
		WorkspaceID: rel.WorkspaceID,
		Role:        rel.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING — user already a member, skip silently
			return nil, nil
		}
		return nil, fmt.Errorf("add user to workspace: %w", err)
	}
	return new(row), nil
}

func (s *pgUserWorkspaceStore) Remove(ctx context.Context, userID, workspaceID int64) error {
	if err := s.queries.RemoveUserFromWorkspace(ctx, generated.RemoveUserFromWorkspaceParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}); err != nil {
		return fmt.Errorf("remove user from workspace: %w", err)
	}
	return nil
}

func (s *pgUserWorkspaceStore) UpdateRole(ctx context.Context, rel *iam.DBUserWorkspace) (*iam.DBUserWorkspace, error) {
	row, err := s.queries.UpdateUserWorkspaceRole(ctx, generated.UpdateUserWorkspaceRoleParams{
		UserID:      rel.UserID,
		WorkspaceID: rel.WorkspaceID,
		Role:        rel.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("update user workspace role: %w", err)
	}
	return new(row), nil
}

func (s *pgUserWorkspaceStore) Get(ctx context.Context, userID, workspaceID int64) (*iam.DBUserWorkspace, error) {
	row, err := s.queries.GetUserWorkspace(ctx, generated.GetUserWorkspaceParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user workspace: %w", err)
	}
	return new(row), nil
}

func (s *pgUserWorkspaceStore) ListByUserID(ctx context.Context, userID int64, q db.ListQuery) (*db.ListResult[iam.DBWorkspaceWithOwnerAndRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountWorkspacesByUserIDParams{
		UserID: userID,
		Status: filterStr(q.Filters, "status"),
		Search: filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountWorkspacesByUserID(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count workspaces by user: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListWorkspacesByUserIDPaginated(ctx, generated.ListWorkspacesByUserIDPaginatedParams{
		UserID:    userID,
		Status:    countParams.Status,
		Search:    countParams.Search,
		SortField: q.SortBy,
		SortOrder: sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspaces by user: %w", err)
	}

	items := make([]iam.DBWorkspaceWithOwnerAndRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBWorkspaceWithOwnerAndRole{
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
			Role:           row.Role,
			JoinedAt:       row.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBWorkspaceWithOwnerAndRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgUserWorkspaceStore) ListByWorkspaceID(ctx context.Context, workspaceID int64, q db.ListQuery) (*db.ListResult[iam.DBUserWithRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountUsersByWorkspaceIDParams{
		WorkspaceID: workspaceID,
		Status:      filterStr(q.Filters, "status"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountUsersByWorkspaceID(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count users by workspace: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListUsersByWorkspaceID(ctx, generated.ListUsersByWorkspaceIDParams{
		WorkspaceID: workspaceID,
		Status:      countParams.Status,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list users by workspace: %w", err)
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
