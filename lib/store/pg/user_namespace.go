package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgUserNamespaceStore struct {
	queries *generated.Queries
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, params store.AddUserNamespaceParams) (*store.UserNamespaceRole, error) {
	row, err := s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      params.UserID,
		NamespaceID: params.NamespaceID,
		Role:        params.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add user to namespace: %w", err)
	}
	return userNamespaceFromRow(row), nil
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

func (s *pgUserNamespaceStore) UpdateRole(ctx context.Context, params store.UpdateRoleParams) (*store.UserNamespaceRole, error) {
	row, err := s.queries.UpdateUserNamespaceRole(ctx, generated.UpdateUserNamespaceRoleParams{
		UserID:      params.UserID,
		NamespaceID: params.NamespaceID,
		Role:        params.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("update user namespace role: %w", err)
	}
	return userNamespaceFromRow(row), nil
}

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*store.UserNamespaceRole, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return userNamespaceFromRow(row), nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]store.NamespaceWithRole, error) {
	rows, err := s.queries.ListNamespacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]store.NamespaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.NamespaceWithRole{
			Namespace: store.Namespace{
				ID:          row.ID,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Description: row.Description,
				OwnerID:     row.OwnerID,
				Visibility:  row.Visibility,
				MaxMembers:  row.MaxMembers,
				Status:      row.Status,
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			Role:     row.Role,
			JoinedAt: toTime(row.JoinedAt),
		})
	}
	return items, nil
}

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]store.UserWithRole, error) {
	rows, err := s.queries.ListUsersByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("list users by namespace: %w", err)
	}

	items := make([]store.UserWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.UserWithRole{
			User: store.User{
				ID:          row.ID,
				Username:    row.Username,
				Email:       row.Email,
				DisplayName: row.DisplayName,
				Phone:       row.Phone,
				AvatarUrl:   row.AvatarUrl,
				Status:      row.Status,
				LastLoginAt: toTimePtr(row.LastLoginAt),
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			Role:     row.Role,
			JoinedAt: toTime(row.JoinedAt),
		})
	}
	return items, nil
}

func userNamespaceFromRow(row generated.UserNamespace) *store.UserNamespaceRole {
	return &store.UserNamespaceRole{
		UserID:      row.UserID,
		NamespaceID: row.NamespaceID,
		Role:        row.Role,
		CreatedAt:   toTime(row.CreatedAt),
	}
}
