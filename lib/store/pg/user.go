package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgUserStore struct {
	queries *generated.Queries
}

func (s *pgUserStore) Create(ctx context.Context, params store.CreateUserParams) (*store.User, error) {
	row, err := s.queries.CreateUser(ctx, generated.CreateUserParams{
		Username:    params.Username,
		Email:       params.Email,
		DisplayName: params.DisplayName,
		Phone:       params.Phone,
		AvatarUrl:   params.AvatarUrl,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByID(ctx context.Context, id int64) (*store.User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) Update(ctx context.Context, params store.UpdateUserParams) (*store.User, error) {
	row, err := s.queries.UpdateUser(ctx, generated.UpdateUserParams{
		ID:          params.ID,
		Username:    params.Username,
		Email:       params.Email,
		DisplayName: params.DisplayName,
		Phone:       params.Phone,
		AvatarUrl:   params.AvatarUrl,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) UpdateLastLogin(ctx context.Context, id int64) error {
	if err := s.queries.UpdateUserLastLogin(ctx, id); err != nil {
		return fmt.Errorf("update user last login: %w", err)
	}
	return nil
}

func (s *pgUserStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *pgUserStore) List(ctx context.Context, params store.ListUsersParams) (*store.ListResult[store.UserWithNamespaces], error) {
	offset, limit := paginationToOffsetLimit(params.Pagination)

	// Escape LIKE params
	var username, email, displayName *string
	if params.Username != nil {
		v := db.EscapeLike(*params.Username)
		username = &v
	}
	if params.Email != nil {
		v := db.EscapeLike(*params.Email)
		email = &v
	}
	if params.DisplayName != nil {
		v := db.EscapeLike(*params.DisplayName)
		displayName = &v
	}

	sortField := params.SortBy
	if sortField == "" {
		sortField = db.UserSortCreatedAt
	}
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = db.SortDesc
	}

	count, err := s.queries.CountUsers(ctx, generated.CountUsersParams{
		Status:      params.Status,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	rows, err := s.queries.ListUsers(ctx, generated.ListUsersParams{
		Status:      params.Status,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		SortField:   sortField,
		SortOrder:   sortOrder,
		PageSize:    limit,
		PageOffset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]store.UserWithNamespaces, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.UserWithNamespaces{
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
			NamespaceNames: row.NamespaceNames,
		})
	}

	return &store.ListResult[store.UserWithNamespaces]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func userFromRow(row generated.User) *store.User {
	return &store.User{
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
	}
}
