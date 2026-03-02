package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

var userListSpec = db.ListSpec{
	Fields: map[string]db.Field{
		"status":       {Column: "u.status", Op: db.Eq},
		"username":     {Column: "u.username", Op: db.Like},
		"email":        {Column: "u.email", Op: db.Like},
		"display_name": {Column: "u.display_name", Op: db.Like},
	},
	DefaultSort: "u.created_at",
}

type pgUserStore struct {
	db      generated.DBTX
	queries *generated.Queries
}

func (s *pgUserStore) Create(ctx context.Context, user *store.User) (*store.User, error) {
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

func (s *pgUserStore) GetByID(ctx context.Context, id int64) (*store.User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &row, nil
}

func (s *pgUserStore) Update(ctx context.Context, user *store.User) (*store.User, error) {
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

func (s *pgUserStore) List(ctx context.Context, q store.ListQuery) (*store.ListResult[store.UserWithNamespaces], error) {
	offset, limit := paginationToOffsetLimit(q.Pagination)
	where, args := db.BuildWhereClause(q.Filters, userListSpec, 1)
	orderBy := db.BuildOrderBy(q.SortBy, q.SortOrder, userListSpec)

	// Count
	var count int64
	countSQL := "SELECT count(DISTINCT u.id) FROM users u" + where
	if err := s.db.QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	// List with LEFT JOIN for namespace names
	n := len(args)
	listSQL := `SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    COALESCE(
        array_agg(DISTINCT ns.name) FILTER (WHERE ns.name IS NOT NULL),
        '{}'
    )::TEXT[] AS namespace_names
FROM users u
LEFT JOIN user_namespaces un ON u.id = un.user_id
LEFT JOIN namespaces ns ON un.namespace_id = ns.id` +
		where +
		` GROUP BY u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
         u.status, u.last_login_at, u.created_at, u.updated_at` +
		orderBy +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", n+1, n+2)

	rows, err := s.db.Query(ctx, listSQL, append(args, limit, offset)...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	items := []store.UserWithNamespaces{}
	for rows.Next() {
		var item store.UserWithNamespaces
		if err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.Email,
			&item.DisplayName,
			&item.Phone,
			&item.AvatarUrl,
			&item.Status,
			&item.LastLoginAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.NamespaceNames,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}

	return &store.ListResult[store.UserWithNamespaces]{
		Items:      items,
		TotalCount: count,
	}, nil
}
