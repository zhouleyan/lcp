package store

import (
	"context"
	"fmt"

	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

var namespaceListSpec = db.ListSpec{
	Fields: map[string]db.Field{
		"status":     {Column: "ns.status", Op: db.Eq},
		"name":       {Column: "ns.name", Op: db.Like},
		"visibility": {Column: "ns.visibility", Op: db.Eq},
		"owner_id":   {Column: "ns.owner_id", Op: db.Eq},
	},
	DefaultSort: "ns.created_at",
}

// ===== pgNamespaceStore =====

type pgNamespaceStore struct {
	db      generated.DBTX
	queries *generated.Queries
}

// NewPGNamespaceStore creates a new PostgreSQL-backed NamespaceStore.
func NewPGNamespaceStore(pool generated.DBTX, queries *generated.Queries) iam.NamespaceStore {
	return &pgNamespaceStore{db: pool, queries: queries}
}

func (s *pgNamespaceStore) Create(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	row, err := s.queries.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        ns.Name,
		DisplayName: ns.DisplayName,
		Description: ns.Description,
		OwnerID:     ns.OwnerID,
		Visibility:  ns.Visibility,
		MaxMembers:  ns.MaxMembers,
		Status:      ns.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// Auto-add owner as member with role "owner"
	_, err = s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      ns.OwnerID,
		NamespaceID: row.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to namespace: %w", err)
	}

	return &row, nil
}

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*iam.DBNamespace, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*iam.DBNamespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespace, error) {
	row, err := s.queries.UpdateNamespace(ctx, generated.UpdateNamespaceParams{
		ID:          ns.ID,
		Name:        ns.Name,
		DisplayName: ns.DisplayName,
		Description: ns.Description,
		OwnerID:     ns.OwnerID,
		Visibility:  ns.Visibility,
		MaxMembers:  ns.MaxMembers,
		Status:      ns.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteNamespace(ctx, id); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (s *pgNamespaceStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBNamespaceWithOwner], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	where, args := db.BuildWhereClause(q.Filters, namespaceListSpec, 1)
	orderBy := db.BuildOrderBy(q.SortBy, q.SortOrder, namespaceListSpec)

	var count int64
	countSQL := "SELECT count(ns.id) FROM namespaces ns" + where
	if err := s.db.QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("count namespaces: %w", err)
	}

	n := len(args)
	listSQL := `SELECT
    ns.id, ns.name, ns.display_name, ns.description, ns.owner_id,
    ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
    u.username AS owner_username
FROM namespaces ns
JOIN users u ON ns.owner_id = u.id` +
		where +
		orderBy +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", n+1, n+2)

	rows, err := s.db.Query(ctx, listSQL, append(args, limit, offset)...)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	defer rows.Close()

	items := []iam.DBNamespaceWithOwner{}
	for rows.Next() {
		var item iam.DBNamespaceWithOwner
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.DisplayName,
			&item.Description,
			&item.OwnerID,
			&item.Visibility,
			&item.MaxMembers,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.OwnerUsername,
		); err != nil {
			return nil, fmt.Errorf("scan namespace row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate namespace rows: %w", err)
	}

	return &db.ListResult[iam.DBNamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

// ===== pgUserNamespaceStore =====

type pgUserNamespaceStore struct {
	queries *generated.Queries
}

// NewPGUserNamespaceStore creates a new PostgreSQL-backed UserNamespaceStore.
func NewPGUserNamespaceStore(queries *generated.Queries) iam.UserNamespaceStore {
	return &pgUserNamespaceStore{queries: queries}
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, rel *iam.DBUserNamespace) (*iam.DBUserNamespace, error) {
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

func (s *pgUserNamespaceStore) UpdateRole(ctx context.Context, rel *iam.DBUserNamespace) (*iam.DBUserNamespace, error) {
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

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*iam.DBUserNamespace, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return &row, nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]iam.DBNamespaceWithRole, error) {
	rows, err := s.queries.ListNamespacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]iam.DBNamespaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, iam.DBNamespaceWithRole{
			Namespace: generated.Namespace{
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

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]iam.DBUserWithRole, error) {
	rows, err := s.queries.ListUsersByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("list users by namespace: %w", err)
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
