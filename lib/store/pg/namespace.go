package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgNamespaceStore struct {
	queries *generated.Queries
}

func (s *pgNamespaceStore) Create(ctx context.Context, params store.CreateNamespaceParams) (*store.Namespace, error) {
	row, err := s.queries.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		OwnerID:     params.OwnerID,
		Visibility:  params.Visibility,
		MaxMembers:  params.MaxMembers,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// Business logic: auto-add owner as member with role "owner"
	_, err = s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      params.OwnerID,
		NamespaceID: row.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to namespace: %w", err)
	}

	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, params store.UpdateNamespaceParams) (*store.Namespace, error) {
	row, err := s.queries.UpdateNamespace(ctx, generated.UpdateNamespaceParams{
		ID:          params.ID,
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		OwnerID:     params.OwnerID,
		Visibility:  params.Visibility,
		MaxMembers:  params.MaxMembers,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteNamespace(ctx, id); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (s *pgNamespaceStore) List(ctx context.Context, params store.ListNamespacesParams) (*store.ListResult[store.NamespaceWithOwner], error) {
	offset, limit := paginationToOffsetLimit(params.Pagination)

	var name *string
	if params.Name != nil {
		v := db.EscapeLike(*params.Name)
		name = &v
	}

	sortField := params.SortBy
	if sortField == "" {
		sortField = db.NamespaceSortCreatedAt
	}
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = db.SortDesc
	}

	count, err := s.queries.CountNamespaces(ctx, generated.CountNamespacesParams{
		Status:     params.Status,
		Name:       name,
		Visibility: params.Visibility,
		OwnerID:    params.OwnerID,
	})
	if err != nil {
		return nil, fmt.Errorf("count namespaces: %w", err)
	}

	rows, err := s.queries.ListNamespaces(ctx, generated.ListNamespacesParams{
		Status:     params.Status,
		Name:       name,
		Visibility: params.Visibility,
		OwnerID:    params.OwnerID,
		SortField:  sortField,
		SortOrder:  sortOrder,
		PageSize:   limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	items := make([]store.NamespaceWithOwner, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.NamespaceWithOwner{
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
			OwnerUsername: row.OwnerUsername,
		})
	}

	return &store.ListResult[store.NamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func namespaceFromRow(row generated.Namespace) *store.Namespace {
	return &store.Namespace{
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
	}
}
