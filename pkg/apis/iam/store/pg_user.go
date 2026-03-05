package store

import (
	"context"
	"fmt"

	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgUserStore struct {
	db      generated.DBTX
	queries *generated.Queries
}

// NewPGUserStore creates a new PostgreSQL-backed UserStore.
func NewPGUserStore(pool generated.DBTX, queries *generated.Queries) iam.UserStore {
	return &pgUserStore{db: pool, queries: queries}
}

func (s *pgUserStore) Create(ctx context.Context, user *iam.DBUser) (*iam.DBUser, error) {
	row, err := s.queries.CreateUser(ctx, generated.CreateUserParams{
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Phone:       user.Phone,
		AvatarUrl:   user.AvatarUrl,
		Status:      user.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) GetByID(ctx context.Context, id int64) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) GetByUsername(ctx context.Context, username string) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) Update(ctx context.Context, user *iam.DBUser) (*iam.DBUser, error) {
	row, err := s.queries.UpdateUser(ctx, generated.UpdateUserParams{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Phone:       user.Phone,
		AvatarUrl:   user.AvatarUrl,
		Status:      user.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) Patch(ctx context.Context, id int64, user *iam.DBUser) (*iam.DBUser, error) {
	row, err := s.queries.PatchUser(ctx, generated.PatchUserParams{
		ID:          id,
		Username:    toNullString(user.Username),
		Email:       toNullString(user.Email),
		DisplayName: toNullString(user.DisplayName),
		Phone:       toNullString(user.Phone),
		AvatarUrl:   toNullString(user.AvatarUrl),
		Status:      toNullString(user.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("patch user: %w", err)
	}
	return &row, nil
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

func (s *pgUserStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	deletedIDs, err := s.queries.DeleteUsersByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete users by ids: %w", err)
	}
	return int64(len(deletedIDs)), nil
}

func (s *pgUserStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBUserWithNamespaces], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	filterStr := func(key string) *string {
		if v, ok := q.Filters[key]; ok {
			if s, ok := v.(string); ok {
				return &s
			}
		}
		return nil
	}

	filterParams := generated.CountUsersParams{
		Status:      filterStr("status"),
		Username:    filterStr("username"),
		Email:       filterStr("email"),
		DisplayName: filterStr("display_name"),
	}

	count, err := s.queries.CountUsers(ctx, filterParams)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListUsers(ctx, generated.ListUsersParams{
		Status:      filterParams.Status,
		Username:    filterParams.Username,
		Email:       filterParams.Email,
		DisplayName: filterParams.DisplayName,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]iam.DBUserWithNamespaces, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBUserWithNamespaces{
			User: generated.User{
				ID:          r.ID,
				Username:    r.Username,
				Email:       r.Email,
				DisplayName: r.DisplayName,
				Phone:       r.Phone,
				AvatarUrl:   r.AvatarUrl,
				Status:      r.Status,
				LastLoginAt: r.LastLoginAt,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			NamespaceNames: r.NamespaceNames,
		})
	}

	return &db.ListResult[iam.DBUserWithNamespaces]{
		Items:      items,
		TotalCount: count,
	}, nil
}
