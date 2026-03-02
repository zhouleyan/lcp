package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
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

type pgNamespaceStore struct {
	db      generated.DBTX
	queries *generated.Queries
}

func (s *pgNamespaceStore) Create(ctx context.Context, ns *store.Namespace) (*store.Namespace, error) {
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

	// Business logic: auto-add owner as member with role "owner"
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

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return &row, nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, ns *store.Namespace) (*store.Namespace, error) {
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

func (s *pgNamespaceStore) List(ctx context.Context, q store.ListQuery) (*store.ListResult[store.NamespaceWithOwner], error) {
	offset, limit := paginationToOffsetLimit(q.Pagination)
	where, args := db.BuildWhereClause(q.Filters, namespaceListSpec, 1)
	orderBy := db.BuildOrderBy(q.SortBy, q.SortOrder, namespaceListSpec)

	// Count
	var count int64
	countSQL := "SELECT count(ns.id) FROM namespaces ns" + where
	if err := s.db.QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("count namespaces: %w", err)
	}

	// List with JOIN for owner username
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

	items := []store.NamespaceWithOwner{}
	for rows.Next() {
		var item store.NamespaceWithOwner
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

	return &store.ListResult[store.NamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}
