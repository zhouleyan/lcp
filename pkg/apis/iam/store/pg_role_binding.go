package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgRoleBindingStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGRoleBindingStore creates a new PostgreSQL-backed RoleBindingStore.
func NewPGRoleBindingStore(pool *pgxpool.Pool, queries *generated.Queries) iam.RoleBindingStore {
	return &pgRoleBindingStore{db: pool, queries: queries}
}

func (s *pgRoleBindingStore) Create(ctx context.Context, rb *iam.DBRoleBinding) (*iam.DBRoleBinding, error) {
	row, err := s.queries.CreateRoleBinding(ctx, generated.CreateRoleBindingParams{
		UserID:      rb.UserID,
		RoleID:      rb.RoleID,
		Scope:       rb.Scope,
		WorkspaceID: rb.WorkspaceID,
		NamespaceID: rb.NamespaceID,
		IsOwner:     rb.IsOwner,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apierrors.NewConflictMessage("role binding already exists")
		}
		return nil, fmt.Errorf("create role binding: %w", err)
	}
	return &row, nil
}

func (s *pgRoleBindingStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteRoleBinding(ctx, id); err != nil {
		return fmt.Errorf("delete role binding: %w", err)
	}
	return nil
}

func (s *pgRoleBindingStore) GetByID(ctx context.Context, id int64) (*iam.DBRoleBinding, error) {
	row, err := s.queries.GetRoleBindingByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("roleBinding", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get role binding by id: %w", err)
	}
	return &row, nil
}

func (s *pgRoleBindingStore) ListPlatform(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBRoleBindingWithDetails], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountRoleBindingsPlatformParams{
		RoleID:  filterInt64(q.Filters, "role_id"),
		IsOwner: filterBool(q.Filters, "is_owner"),
		Search:  filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoleBindingsPlatform(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count platform role bindings: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoleBindingsPlatform(ctx, generated.ListRoleBindingsPlatformParams{
		RoleID:     countParams.RoleID,
		IsOwner:    countParams.IsOwner,
		Search:     countParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list platform role bindings: %w", err)
	}

	items := make([]iam.DBRoleBindingWithDetails, 0, len(rows))
	for _, r := range rows {
		items = append(items, rowToRoleBindingWithDetails(
			r.ID, r.UserID, r.RoleID, r.Scope, r.WorkspaceID, r.NamespaceID, r.IsOwner, r.CreatedAt,
			r.Username, r.UserDisplayName, r.RoleName, r.RoleDisplayName,
		))
	}

	return &db.ListResult[iam.DBRoleBindingWithDetails]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListByWorkspaceID(ctx context.Context, workspaceID int64, q db.ListQuery) (*db.ListResult[iam.DBRoleBindingWithDetails], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	wsID := &workspaceID

	countParams := generated.CountRoleBindingsByWorkspaceIDParams{
		WorkspaceID: wsID,
		RoleID:      filterInt64(q.Filters, "role_id"),
		IsOwner:     filterBool(q.Filters, "is_owner"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoleBindingsByWorkspaceID(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count workspace role bindings: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoleBindingsByWorkspaceID(ctx, generated.ListRoleBindingsByWorkspaceIDParams{
		WorkspaceID: wsID,
		RoleID:      countParams.RoleID,
		IsOwner:     countParams.IsOwner,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace role bindings: %w", err)
	}

	items := make([]iam.DBRoleBindingWithDetails, 0, len(rows))
	for _, r := range rows {
		items = append(items, rowToRoleBindingWithDetails(
			r.ID, r.UserID, r.RoleID, r.Scope, r.WorkspaceID, r.NamespaceID, r.IsOwner, r.CreatedAt,
			r.Username, r.UserDisplayName, r.RoleName, r.RoleDisplayName,
		))
	}

	return &db.ListResult[iam.DBRoleBindingWithDetails]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListByNamespaceID(ctx context.Context, namespaceID int64, q db.ListQuery) (*db.ListResult[iam.DBRoleBindingWithDetails], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	nsID := &namespaceID

	countParams := generated.CountRoleBindingsByNamespaceIDParams{
		NamespaceID: nsID,
		RoleID:      filterInt64(q.Filters, "role_id"),
		IsOwner:     filterBool(q.Filters, "is_owner"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoleBindingsByNamespaceID(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count namespace role bindings: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoleBindingsByNamespaceID(ctx, generated.ListRoleBindingsByNamespaceIDParams{
		NamespaceID: nsID,
		RoleID:      countParams.RoleID,
		IsOwner:     countParams.IsOwner,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespace role bindings: %w", err)
	}

	items := make([]iam.DBRoleBindingWithDetails, 0, len(rows))
	for _, r := range rows {
		items = append(items, rowToRoleBindingWithDetails(
			r.ID, r.UserID, r.RoleID, r.Scope, r.WorkspaceID, r.NamespaceID, r.IsOwner, r.CreatedAt,
			r.Username, r.UserDisplayName, r.RoleName, r.RoleDisplayName,
		))
	}

	return &db.ListResult[iam.DBRoleBindingWithDetails]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListByUserID(ctx context.Context, userID int64, q db.ListQuery) (*db.ListResult[iam.DBRoleBindingWithDetails], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountRoleBindingsByUserIDParams{
		UserID: userID,
		Scope:  filterStr(q.Filters, "scope"),
		RoleID: filterInt64(q.Filters, "role_id"),
		Search: filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoleBindingsByUserID(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count user role bindings: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoleBindingsByUserID(ctx, generated.ListRoleBindingsByUserIDParams{
		UserID:     userID,
		Scope:      countParams.Scope,
		RoleID:     countParams.RoleID,
		Search:     countParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list user role bindings: %w", err)
	}

	items := make([]iam.DBRoleBindingWithDetails, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBRoleBindingWithDetails{
			RoleBinding: generated.RoleBinding{
				ID:          r.ID,
				UserID:      r.UserID,
				RoleID:      r.RoleID,
				Scope:       r.Scope,
				WorkspaceID: r.WorkspaceID,
				NamespaceID: r.NamespaceID,
				IsOwner:     r.IsOwner,
				CreatedAt:   r.CreatedAt,
			},
			RoleName:        r.RoleName,
			RoleDisplayName: r.RoleDisplayName,
		})
	}

	return &db.ListResult[iam.DBRoleBindingWithDetails]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) CountByRoleAndScope(ctx context.Context, roleID int64, scope string) (int64, error) {
	count, err := s.queries.CountRoleBindingsByRoleAndScope(ctx, generated.CountRoleBindingsByRoleAndScopeParams{
		RoleID: roleID,
		Scope:  scope,
	})
	if err != nil {
		return 0, fmt.Errorf("count role bindings by role and scope: %w", err)
	}
	return count, nil
}

func (s *pgRoleBindingStore) GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	ptrs, err := s.queries.GetAccessibleWorkspaceIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get accessible workspace ids: %w", err)
	}
	ids := make([]int64, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			ids = append(ids, *p)
		}
	}
	return ids, nil
}

func (s *pgRoleBindingStore) GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	ptrs, err := s.queries.GetAccessibleNamespaceIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get accessible namespace ids: %w", err)
	}
	ids := make([]int64, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			ids = append(ids, *p)
		}
	}
	return ids, nil
}

func (s *pgRoleBindingStore) GetUserIDsByWorkspaceID(ctx context.Context, workspaceID int64) ([]int64, error) {
	wsID := &workspaceID
	ids, err := s.queries.GetUserIDsByWorkspaceID(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("get user ids by workspace: %w", err)
	}
	return ids, nil
}

func (s *pgRoleBindingStore) GetUserIDsByNamespaceID(ctx context.Context, namespaceID int64) ([]int64, error) {
	nsID := &namespaceID
	ids, err := s.queries.GetUserIDsByNamespaceID(ctx, nsID)
	if err != nil {
		return nil, fmt.Errorf("get user ids by namespace: %w", err)
	}
	return ids, nil
}

func (s *pgRoleBindingStore) LoadUserPermissionRules(ctx context.Context, userID int64) ([]iam.UserPermissionRuleRow, error) {
	rows, err := s.queries.LoadUserPermissionRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user permission rules: %w", err)
	}
	items := make([]iam.UserPermissionRuleRow, len(rows))
	for i, r := range rows {
		items[i] = iam.UserPermissionRuleRow{
			Scope:       r.Scope,
			WorkspaceID: r.WorkspaceID,
			NamespaceID: r.NamespaceID,
			Pattern:     r.Pattern,
		}
	}
	return items, nil
}

func (s *pgRoleBindingStore) GetUserRoleBindingsWithRules(ctx context.Context, userID int64) ([]iam.UserRoleBindingWithRules, error) {
	rows, err := s.queries.GetUserRoleBindingsWithRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user role bindings with rules: %w", err)
	}
	items := make([]iam.UserRoleBindingWithRules, len(rows))
	for i, r := range rows {
		items[i] = iam.UserRoleBindingWithRules{
			Scope:       r.Scope,
			WorkspaceID: r.WorkspaceID,
			NamespaceID: r.NamespaceID,
			RoleName:    r.RoleName,
			Pattern:     r.Pattern,
		}
	}
	return items, nil
}

// rowToRoleBindingWithDetails converts individual fields to DBRoleBindingWithDetails.
func rowToRoleBindingWithDetails(
	id, userID, roleID int64, scope string, workspaceID, namespaceID *int64, isOwner bool, createdAt time.Time,
	username, userDisplayName, roleName, roleDisplayName string,
) iam.DBRoleBindingWithDetails {
	return iam.DBRoleBindingWithDetails{
		RoleBinding: generated.RoleBinding{
			ID:          id,
			UserID:      userID,
			RoleID:      roleID,
			Scope:       scope,
			WorkspaceID: workspaceID,
			NamespaceID: namespaceID,
			IsOwner:     isOwner,
			CreatedAt:   createdAt,
		},
		Username:        username,
		UserDisplayName: userDisplayName,
		RoleName:        roleName,
		RoleDisplayName: roleDisplayName,
	}
}
