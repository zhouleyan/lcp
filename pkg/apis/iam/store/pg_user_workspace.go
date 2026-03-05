package store

import (
	"fmt"

	"context"

	"lcp.io/lcp/pkg/apis/iam"
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
		return nil, fmt.Errorf("add user to workspace: %w", err)
	}
	return &row, nil
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
	return &row, nil
}

func (s *pgUserWorkspaceStore) Get(ctx context.Context, userID, workspaceID int64) (*iam.DBUserWorkspace, error) {
	row, err := s.queries.GetUserWorkspace(ctx, generated.GetUserWorkspaceParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user workspace: %w", err)
	}
	return &row, nil
}

func (s *pgUserWorkspaceStore) ListByUserID(ctx context.Context, userID int64) ([]iam.DBWorkspaceWithRole, error) {
	rows, err := s.queries.ListWorkspacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces by user: %w", err)
	}

	items := make([]iam.DBWorkspaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBWorkspaceWithRole{
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
			Role:     row.Role,
			JoinedAt: row.JoinedAt,
		})
	}
	return items, nil
}

func (s *pgUserWorkspaceStore) ListByWorkspaceID(ctx context.Context, workspaceID int64) ([]iam.DBUserWithRole, error) {
	rows, err := s.queries.ListUsersByWorkspaceID(ctx, workspaceID)
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
	return items, nil
}
