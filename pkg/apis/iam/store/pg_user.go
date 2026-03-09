package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgUserStore struct {
	queries *generated.Queries
}

// NewPGUserStore creates a new PostgreSQL-backed UserStore.
func NewPGUserStore(queries *generated.Queries) iam.UserStore {
	return &pgUserStore{queries: queries}
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
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("user", user.Username)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return new(createRowToUser(row)), nil
}

func (s *pgUserStore) GetByID(ctx context.Context, id int64) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return new(getByIDRowToUser(row)), nil
}

func (s *pgUserStore) GetByUsername(ctx context.Context, username string) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", username)
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return new(getByUsernameRowToUser(row)), nil
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", email)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return new(getByEmailRowToUser(row)), nil
}

func (s *pgUserStore) GetByPhone(ctx context.Context, phone string) (*iam.DBUser, error) {
	row, err := s.queries.GetUserByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", phone)
		}
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return new(getByPhoneRowToUser(row)), nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", fmt.Sprintf("%d", user.ID))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("user", user.Username)
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	return new(updateRowToUser(row)), nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", fmt.Sprintf("%d", id))
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("user", user.Username)
		}
		return nil, fmt.Errorf("patch user: %w", err)
	}
	return new(patchRowToUser(row)), nil
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

	filterParams := generated.CountUsersParams{
		Status: filterStr(q.Filters, "status"),
		Search: filterStr(q.Filters, "search"),
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
		Status:     filterParams.Status,
		Search:     filterParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
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

func (s *pgUserStore) GetUserForAuth(ctx context.Context, identifier string) (*iam.DBUserForAuth, error) {
	row, err := s.queries.GetUserForAuth(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("user", identifier)
		}
		return nil, fmt.Errorf("get user for auth: %w", err)
	}
	return new(row), nil
}

func (s *pgUserStore) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	if err := s.queries.SetPasswordHash(ctx, generated.SetPasswordHashParams{
		ID:           id,
		PasswordHash: hash,
	}); err != nil {
		return fmt.Errorf("set password hash: %w", err)
	}
	return nil
}

// Row-to-User conversion helpers. These exist because sqlc generates separate
// Row types when the query's column list doesn't match the full table schema
// (the users table has password_hash which is excluded from standard queries).

func createRowToUser(r generated.CreateUserRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func getByIDRowToUser(r generated.GetUserByIDRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func getByUsernameRowToUser(r generated.GetUserByUsernameRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func getByEmailRowToUser(r generated.GetUserByEmailRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func getByPhoneRowToUser(r generated.GetUserByPhoneRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func updateRowToUser(r generated.UpdateUserRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func patchRowToUser(r generated.PatchUserRow) generated.User {
	return generated.User{
		ID: r.ID, Username: r.Username, Email: r.Email,
		DisplayName: r.DisplayName, Phone: r.Phone, AvatarUrl: r.AvatarUrl,
		Status: r.Status, LastLoginAt: r.LastLoginAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}
