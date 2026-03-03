package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db/generated"
	libstore "lcp.io/lcp/lib/store"

	nsstore "lcp.io/lcp/pkg/modules/namespace/store"
)

type pgUserNamespaceStore struct {
	queries *generated.Queries
}

// NewUserNamespaceStore creates a new PostgreSQL-backed UserNamespaceStore.
func NewUserNamespaceStore(queries *generated.Queries) nsstore.UserNamespaceStore {
	return &pgUserNamespaceStore{queries: queries}
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, rel *nsstore.UserNamespaceRole) (*nsstore.UserNamespaceRole, error) {
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

func (s *pgUserNamespaceStore) UpdateRole(ctx context.Context, rel *nsstore.UserNamespaceRole) (*nsstore.UserNamespaceRole, error) {
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

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*nsstore.UserNamespaceRole, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return &row, nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]nsstore.NamespaceWithRole, error) {
	rows, err := s.queries.ListNamespacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]nsstore.NamespaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, nsstore.NamespaceWithRole{
			Namespace: libstore.Namespace{
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

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]nsstore.UserWithRole, error) {
	rows, err := s.queries.ListUsersByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("list users by namespace: %w", err)
	}

	items := make([]nsstore.UserWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, nsstore.UserWithRole{
			User: libstore.User{
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
