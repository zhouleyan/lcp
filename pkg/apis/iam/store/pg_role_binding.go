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
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflictMessage("role binding already exists")
		}
		return nil, fmt.Errorf("create role binding: %w", err)
	}
	return new(row), nil
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
	return new(row), nil
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
		item := iam.DBRoleBindingWithDetails{
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
		}
		if r.WorkspaceName != nil {
			item.WorkspaceName = *r.WorkspaceName
		}
		if r.NamespaceName != nil {
			item.NamespaceName = *r.NamespaceName
		}
		items = append(items, item)
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

func (s *pgRoleBindingStore) TransferOwnership(ctx context.Context, scope string, resourceID int64, callerID int64, callerIsPlatformAdmin bool, newOwnerUserID int64, adminRoleName string) (int64, error) {
	var scopeColumn string
	switch scope {
	case iam.ScopeWorkspace:
		scopeColumn = "workspace_id"
	case iam.ScopeNamespace:
		scopeColumn = "namespace_id"
	default:
		return 0, fmt.Errorf("unsupported scope for ownership transfer: %s", scope)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Find current owner (with row lock)
	var oldOwnerUserID int64
	findOwnerQuery := fmt.Sprintf(
		"SELECT user_id FROM role_bindings WHERE scope = $1 AND %s = $2 AND is_owner = true FOR UPDATE",
		scopeColumn,
	)
	if err := tx.QueryRow(ctx, findOwnerQuery, scope, resourceID).Scan(&oldOwnerUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apierrors.NewNotFound("owner", fmt.Sprintf("scope=%s, id=%d", scope, resourceID))
		}
		return 0, fmt.Errorf("find current owner: %w", err)
	}

	// 2. Authorization: caller must be current owner or platform admin
	if !callerIsPlatformAdmin && callerID != oldOwnerUserID {
		return 0, apierrors.NewForbidden("only the current owner or platform admin can transfer ownership")
	}

	// 3. Verify new owner is a member (has any binding in this scope+resource)
	memberQuery := fmt.Sprintf(
		"SELECT EXISTS(SELECT 1 FROM role_bindings WHERE user_id = $1 AND scope = $2 AND %s = $3)",
		scopeColumn,
	)
	var isMember bool
	if err := tx.QueryRow(ctx, memberQuery, newOwnerUserID, scope, resourceID).Scan(&isMember); err != nil {
		return 0, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return 0, apierrors.NewBadRequest("new owner must be a member of the resource", nil)
	}

	// 4. Clear current ownership
	clearQuery := fmt.Sprintf(
		"UPDATE role_bindings SET is_owner = false WHERE scope = $1 AND %s = $2 AND is_owner = true",
		scopeColumn,
	)
	if _, err := tx.Exec(ctx, clearQuery, scope, resourceID); err != nil {
		return 0, fmt.Errorf("clear current ownership: %w", err)
	}

	// 5. Look up admin role ID (scoped to workspace or namespace)
	var adminRoleID int64
	var adminRoleQuery string
	if scope == iam.ScopeWorkspace {
		adminRoleQuery = "SELECT id FROM roles WHERE name = $1 AND workspace_id = $2"
	} else {
		adminRoleQuery = "SELECT id FROM roles WHERE name = $1 AND namespace_id = $2"
	}
	if err := tx.QueryRow(ctx, adminRoleQuery, adminRoleName, resourceID).Scan(&adminRoleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("admin role %q not found", adminRoleName)
		}
		return 0, fmt.Errorf("look up admin role: %w", err)
	}

	// 6. Upsert owner binding for new owner (admin role + is_owner=true)
	if scope == iam.ScopeNamespace {
		// For namespace scope, we also need workspace_id
		var wsID int64
		if err := tx.QueryRow(ctx, "SELECT workspace_id FROM namespaces WHERE id = $1", resourceID).Scan(&wsID); err != nil {
			return 0, fmt.Errorf("look up namespace workspace: %w", err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, namespace_id, is_owner)
			 VALUES ($1, $2, $3, $4, $5, true)
			 ON CONFLICT (user_id, role_id, namespace_id) WHERE scope = 'namespace'
			 DO UPDATE SET is_owner = true`,
			newOwnerUserID, adminRoleID, scope, wsID, resourceID,
		)
	} else {
		_, err = tx.Exec(ctx,
			`INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, is_owner)
			 VALUES ($1, $2, $3, $4, true)
			 ON CONFLICT (user_id, role_id, workspace_id) WHERE scope = 'workspace'
			 DO UPDATE SET is_owner = true`,
			newOwnerUserID, adminRoleID, scope, resourceID,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("upsert owner binding: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return oldOwnerUserID, nil
}

func (s *pgRoleBindingStore) AddWorkspaceMember(ctx context.Context, userID, workspaceID int64, roleID int64) error {
	wsIDPtr := &workspaceID
	bindRoleID := roleID
	if bindRoleID == 0 {
		viewerRole, err := s.queries.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
			Name:        iam.RoleWorkspaceViewer,
			WorkspaceID: wsIDPtr,
		})
		if err != nil {
			return fmt.Errorf("get workspace-viewer role: %w", err)
		}
		bindRoleID = viewerRole.ID
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Remove existing non-owner bindings so the new role replaces the old one
	if _, err := qtx.DeleteNonOwnerWorkspaceBindings(ctx, generated.DeleteNonOwnerWorkspaceBindingsParams{
		UserID:      userID,
		WorkspaceID: wsIDPtr,
	}); err != nil {
		return fmt.Errorf("delete old workspace bindings: %w", err)
	}

	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      userID,
		RoleID:      bindRoleID,
		Scope:       iam.ScopeWorkspace,
		WorkspaceID: wsIDPtr,
	}); err != nil {
		return fmt.Errorf("add workspace member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *pgRoleBindingStore) AddNamespaceMember(ctx context.Context, userID, namespaceID int64, roleID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Get namespace to find workspace_id
	var wsID int64
	if err := tx.QueryRow(ctx, "SELECT workspace_id FROM namespaces WHERE id = $1", namespaceID).Scan(&wsID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NewNotFound("namespace", fmt.Sprintf("%d", namespaceID))
		}
		return fmt.Errorf("get namespace workspace: %w", err)
	}

	wsIDPtr := &wsID
	wsMemberRole, err := qtx.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
		Name:        iam.RoleWorkspaceMember,
		WorkspaceID: wsIDPtr,
	})
	if err != nil {
		return fmt.Errorf("get workspace-member role: %w", err)
	}

	bindRoleID := roleID
	if bindRoleID == 0 {
		nsIDPtr := &namespaceID
		nsViewerRole, err := qtx.GetRoleByNameAndNamespace(ctx, generated.GetRoleByNameAndNamespaceParams{
			Name:        iam.RoleNamespaceViewer,
			NamespaceID: nsIDPtr,
		})
		if err != nil {
			return fmt.Errorf("get namespace-viewer role: %w", err)
		}
		bindRoleID = nsViewerRole.ID
	}

	// Auto-add to workspace with minimal membership role (no workspace-level permissions)
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      userID,
		RoleID:      wsMemberRole.ID,
		Scope:       iam.ScopeWorkspace,
		WorkspaceID: wsIDPtr,
	}); err != nil {
		return fmt.Errorf("auto-add workspace member: %w", err)
	}

	nsIDPtr := &namespaceID

	// Remove existing non-owner namespace bindings so the new role replaces the old one
	if _, err := qtx.DeleteNonOwnerNamespaceBindings(ctx, generated.DeleteNonOwnerNamespaceBindingsParams{
		UserID:      userID,
		NamespaceID: nsIDPtr,
	}); err != nil {
		return fmt.Errorf("delete old namespace bindings: %w", err)
	}

	// Add to namespace
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      userID,
		RoleID:      bindRoleID,
		Scope:       iam.ScopeNamespace,
		WorkspaceID: wsIDPtr,
		NamespaceID: nsIDPtr,
	}); err != nil {
		return fmt.Errorf("add namespace member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *pgRoleBindingStore) RemoveWorkspaceMember(ctx context.Context, userID, workspaceID int64) error {
	wsID := &workspaceID
	n, err := s.queries.DeleteNonOwnerWorkspaceBindings(ctx, generated.DeleteNonOwnerWorkspaceBindingsParams{
		UserID:      userID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}
	if n == 0 {
		return apierrors.NewBadRequest("cannot remove workspace owner", nil)
	}
	return nil
}

func (s *pgRoleBindingStore) RemoveNamespaceMember(ctx context.Context, userID, namespaceID int64) error {
	nsID := &namespaceID
	n, err := s.queries.DeleteNonOwnerNamespaceBindings(ctx, generated.DeleteNonOwnerNamespaceBindingsParams{
		UserID:      userID,
		NamespaceID: nsID,
	})
	if err != nil {
		return fmt.Errorf("remove namespace member: %w", err)
	}
	if n == 0 {
		return apierrors.NewBadRequest("cannot remove namespace owner", nil)
	}
	return nil
}

func (s *pgRoleBindingStore) ListWorkspaceMembers(ctx context.Context, workspaceID int64, q db.ListQuery) (*db.ListResult[iam.DBUserWithRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	wsID := &workspaceID

	countParams := generated.CountWorkspaceMembersParams{
		WorkspaceID: wsID,
		Status:      filterStr(q.Filters, "status"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountWorkspaceMembers(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count workspace members: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListWorkspaceMembers(ctx, generated.ListWorkspaceMembersParams{
		WorkspaceID: wsID,
		Status:      countParams.Status,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}

	items := make([]iam.DBUserWithRole, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBUserWithRole{
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
			Role:     r.RoleName,
			JoinedAt: r.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBUserWithRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListNamespaceMembers(ctx context.Context, namespaceID int64, q db.ListQuery) (*db.ListResult[iam.DBUserWithRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)
	nsID := &namespaceID

	countParams := generated.CountNamespaceMembersParams{
		NamespaceID: nsID,
		Status:      filterStr(q.Filters, "status"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountNamespaceMembers(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count namespace members: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListNamespaceMembers(ctx, generated.ListNamespaceMembersParams{
		NamespaceID: nsID,
		Status:      countParams.Status,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespace members: %w", err)
	}

	items := make([]iam.DBUserWithRole, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBUserWithRole{
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
			Role:     r.RoleName,
			JoinedAt: r.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBUserWithRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListUserWorkspaces(ctx context.Context, userID int64, q db.ListQuery) (*db.ListResult[iam.DBWorkspaceWithOwnerAndRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountUserWorkspacesParams{
		UserID: userID,
		Status: filterStr(q.Filters, "status"),
		Search: filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountUserWorkspaces(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count user workspaces: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListUserWorkspaces(ctx, generated.ListUserWorkspacesParams{
		UserID:     userID,
		Status:     countParams.Status,
		Search:     countParams.Search,
		SortField:  q.SortBy,
		SortOrder:  sortOrder,
		PageOffset: offset,
		PageSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list user workspaces: %w", err)
	}

	items := make([]iam.DBWorkspaceWithOwnerAndRole, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBWorkspaceWithOwnerAndRole{
			Workspace: generated.Workspace{
				ID:          r.ID,
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
				OwnerID:     r.OwnerID,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			OwnerUsername:   r.OwnerUsername,
			NamespaceCount:  r.NamespaceCount,
			MemberCount:     r.MemberCount,
			Role:            r.RoleName,
			RoleDisplayName: r.RoleDisplayName,
			JoinedAt:        r.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBWorkspaceWithOwnerAndRole]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func (s *pgRoleBindingStore) ListUserNamespaces(ctx context.Context, userID int64, q db.ListQuery) (*db.ListResult[iam.DBNamespaceWithOwnerAndRole], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountUserNamespacesParams{
		UserID:      userID,
		Status:      filterStr(q.Filters, "status"),
		Visibility:  filterStr(q.Filters, "visibility"),
		WorkspaceID: filterInt64(q.Filters, "workspace_id"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountUserNamespaces(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count user namespaces: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListUserNamespaces(ctx, generated.ListUserNamespacesParams{
		UserID:      userID,
		Status:      countParams.Status,
		Visibility:  countParams.Visibility,
		WorkspaceID: countParams.WorkspaceID,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list user namespaces: %w", err)
	}

	items := make([]iam.DBNamespaceWithOwnerAndRole, 0, len(rows))
	for _, r := range rows {
		items = append(items, iam.DBNamespaceWithOwnerAndRole{
			Namespace: generated.Namespace{
				ID:          r.ID,
				Name:        r.Name,
				DisplayName: r.DisplayName,
				Description: r.Description,
				WorkspaceID: r.WorkspaceID,
				OwnerID:     r.OwnerID,
				Visibility:  r.Visibility,
				MaxMembers:  r.MaxMembers,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			OwnerUsername:   r.OwnerUsername,
			WorkspaceName:   r.WorkspaceName,
			MemberCount:     r.MemberCount,
			Role:            r.RoleName,
			RoleDisplayName: r.RoleDisplayName,
			JoinedAt:        r.JoinedAt,
		})
	}

	return &db.ListResult[iam.DBNamespaceWithOwnerAndRole]{
		Items:      items,
		TotalCount: count,
	}, nil
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
