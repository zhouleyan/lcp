package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// PasswordHasher 密码哈希函数类型。
type PasswordHasher func(password string) (string, error)

// ===== userStorage 用户存储 =====

// userStorage 用户资源的 REST 存储实现，支持 CRUD、批量删除和密码管理。
type userStorage struct {
	dbStore    UserStore
	hashPasswd PasswordHasher
}

// NewUserStorage 创建用户 REST 存储（无密码功能）。
func NewUserStorage(dbStore UserStore) rest.StandardStorage {
	return &userStorage{dbStore: dbStore}
}

// NewUserStorageWithPassword 创建支持密码哈希的用户 REST 存储。
func NewUserStorageWithPassword(dbStore UserStore, hashPasswd PasswordHasher) rest.StandardStorage {
	return &userStorage{dbStore: dbStore, hashPasswd: hashPasswd}
}

func (s *userStorage) NewObject() runtime.Object { return &User{} }

// Get 获取用户详情。
// +openapi:summary=获取用户详情
func (s *userStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["userId"]
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	user, err := s.dbStore.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return userToAPI(user), nil
}

// List 获取用户列表，支持分页、排序和筛选。
// +openapi:summary=获取用户列表
func (s *userStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}

	result, err := s.dbStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(result.Items))
	for i, item := range result.Items {
		items[i] = *userWithNamespacesToAPI(&item)
	}

	return &UserList{
		TypeMeta:   runtime.TypeMeta{Kind: "UserList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create 创建用户。如果提供了密码，会进行密码策略验证并使用 bcrypt 哈希存储。
// +openapi:summary=创建用户
func (s *userStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	user, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", obj)
	}

	if errs := ValidateUserCreate(&user.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	// Validate and hash password if provided
	if user.Spec.Password != "" {
		if errs := ValidatePassword(user.Spec.Password); errs.HasErrors() {
			return nil, apierrors.NewBadRequest("validation failed", errs)
		}
	}

	if options.DryRun {
		user.Spec.Password = ""
		return user, nil
	}

	created, err := s.dbStore.Create(ctx, &DBUser{
		Username:    user.Spec.Username,
		Email:       user.Spec.Email,
		DisplayName: user.Spec.DisplayName,
		Phone:       user.Spec.Phone,
		AvatarUrl:   user.Spec.AvatarURL,
		Status:      user.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	// Hash and store password after user creation
	if user.Spec.Password != "" && s.hashPasswd != nil {
		hash, err := s.hashPasswd(user.Spec.Password)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("hash password: %w", err))
		}
		if err := s.dbStore.SetPasswordHash(ctx, created.ID, hash); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("set password: %w", err))
		}
	}

	return userToAPI(created), nil
}

// Update 全量更新用户信息。
// +openapi:summary=更新用户信息（全量）
func (s *userStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	user, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", obj)
	}

	if errs := ValidateUserUpdate(&user.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return user, nil
	}

	id := options.PathParams["userId"]
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	updated, err := s.dbStore.Update(ctx, &DBUser{
		ID:          uid,
		Username:    user.Spec.Username,
		Email:       user.Spec.Email,
		DisplayName: user.Spec.DisplayName,
		Phone:       user.Spec.Phone,
		AvatarUrl:   user.Spec.AvatarURL,
		Status:      user.Spec.Status,
	})
	if err != nil {
		return nil, err
	}
	return userToAPI(updated), nil
}

// Patch 部分更新用户信息，仅更新请求中提供的字段。
// +openapi:summary=更新用户信息（部分）
func (s *userStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	user, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", obj)
	}

	id := options.PathParams["userId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	patched, err := s.dbStore.Patch(ctx, uid, &DBUser{
		Username:    user.Spec.Username,
		Email:       user.Spec.Email,
		DisplayName: user.Spec.DisplayName,
		Phone:       user.Spec.Phone,
		AvatarUrl:   user.Spec.AvatarURL,
		Status:      user.Spec.Status,
	})
	if err != nil {
		return nil, err
	}
	return userToAPI(patched), nil
}

// Delete 删除单个用户。
// +openapi:summary=删除用户
func (s *userStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["userId"]
	uid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	if err := s.dbStore.Delete(ctx, uid); err != nil {
		return err
	}
	return nil
}

// DeleteCollection 批量删除用户。
// +openapi:summary=批量删除用户
func (s *userStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		uid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, uid)
	}

	count, err := s.dbStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== workspaceStorage 工作空间存储 =====

// workspaceStorage 工作空间资源的 REST 存储实现。创建工作空间时自动创建默认项目并添加所有者为成员。
type workspaceStorage struct {
	wsStore   WorkspaceStore
	userStore UserStore
	rbStore   RoleBindingStore
	checker   PermissionChecker
}

// NewWorkspaceStorage 创建工作空间 REST 存储。
func NewWorkspaceStorage(wsStore WorkspaceStore, userStore UserStore, rbStore RoleBindingStore, checker PermissionChecker) rest.StandardStorage {
	return &workspaceStorage{wsStore: wsStore, userStore: userStore, rbStore: rbStore, checker: checker}
}

func (s *workspaceStorage) NewObject() runtime.Object { return &Workspace{} }

// +openapi:summary=获取工作空间详情
func (s *workspaceStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["workspaceId"]
	wid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", id), nil)
	}

	ws, err := s.wsStore.GetByID(ctx, wid)
	if err != nil {
		return nil, err
	}
	return workspaceWithOwnerToAPI(ws), nil
}

// +openapi:summary=获取工作空间列表
func (s *workspaceStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}

	// Inject access filter for non-admin users
	if af := filters.AccessFilterFromContext(ctx); af != nil && af.WorkspaceIDs != nil {
		query.Filters["accessible_ids"] = af.WorkspaceIDs
	}

	result, err := s.wsStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Workspace, len(result.Items))
	for i, item := range result.Items {
		items[i] = *workspaceWithOwnerToAPI(&item)
	}

	return &WorkspaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "WorkspaceList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=创建工作空间
func (s *workspaceStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ws, ok := obj.(*Workspace)
	if !ok {
		return nil, fmt.Errorf("expected *Workspace, got %T", obj)
	}

	// Auto-inject ownerId from authenticated user
	userID, ok := oidc.UserIDFromContext(ctx)
	if ok {
		ws.Spec.OwnerID = strconv.FormatInt(userID, 10)
	}

	if errs := ValidateWorkspaceCreate(ws.ObjectMeta.Name, &ws.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ws, nil
	}

	ownerID, err := parseID(ws.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ws.Spec.OwnerID), nil)
	}

	// Check owner exists
	if _, err := s.userStore.GetByID(ctx, ownerID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("owner user %d not found", ownerID), nil)
	}

	status := ws.Spec.Status
	if status == "" {
		status = "active"
	}

	created, err := s.wsStore.Create(ctx, &DBWorkspace{
		Name:        ws.ObjectMeta.Name,
		DisplayName: ws.Spec.DisplayName,
		Description: ws.Spec.Description,
		OwnerID:     ownerID,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	return workspaceWithOwnerToAPI(created), nil
}

// +openapi:summary=更新工作空间信息（全量）
func (s *workspaceStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ws, ok := obj.(*Workspace)
	if !ok {
		return nil, fmt.Errorf("expected *Workspace, got %T", obj)
	}

	if errs := ValidateWorkspaceUpdate(&ws.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ws, nil
	}

	id := options.PathParams["workspaceId"]
	wid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", id), nil)
	}

	ownerID, err := parseID(ws.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ws.Spec.OwnerID), nil)
	}

	updated, err := s.wsStore.Update(ctx, &DBWorkspace{
		ID:          wid,
		Name:        ws.ObjectMeta.Name,
		DisplayName: ws.Spec.DisplayName,
		Description: ws.Spec.Description,
		OwnerID:     ownerID,
		Status:      ws.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return workspaceToAPI(updated), nil
}

// +openapi:summary=更新工作空间信息（部分）
func (s *workspaceStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	ws, ok := obj.(*Workspace)
	if !ok {
		return nil, fmt.Errorf("expected *Workspace, got %T", obj)
	}

	id := options.PathParams["workspaceId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	wid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", id), nil)
	}

	var ownerID int64
	if ws.Spec.OwnerID != "" {
		ownerID, err = parseID(ws.Spec.OwnerID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ws.Spec.OwnerID), nil)
		}
	}

	patched, err := s.wsStore.Patch(ctx, wid, &DBWorkspace{
		Name:        ws.ObjectMeta.Name,
		DisplayName: ws.Spec.DisplayName,
		Description: ws.Spec.Description,
		OwnerID:     ownerID,
		Status:      ws.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return workspaceToAPI(patched), nil
}

// +openapi:summary=删除工作空间
func (s *workspaceStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["workspaceId"]
	wid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", id), nil)
	}

	// Collect affected user IDs before deletion (CASCADE will remove role_bindings)
	var affectedUserIDs []int64
	if s.rbStore != nil {
		affectedUserIDs, _ = s.rbStore.GetUserIDsByWorkspaceID(ctx, wid)
	}

	if err := s.wsStore.Delete(ctx, wid); err != nil {
		return err
	}

	// Invalidate permission cache for affected users
	if s.checker != nil {
		for _, uid := range affectedUserIDs {
			s.checker.InvalidateCache(uid)
		}
	}
	return nil
}

// +openapi:summary=批量删除工作空间
func (s *workspaceStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		wid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, wid)
	}

	count, err := s.wsStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== namespaceStorage 项目存储 =====

// namespaceStorage 项目资源的 REST 存储实现。支持按工作空间筛选项目列表。
// +openapi:path=/workspaces/{workspaceId}/namespaces
type namespaceStorage struct {
	nsStore   NamespaceStore
	wsStore   WorkspaceStore
	userStore UserStore
	rbStore   RoleBindingStore
	checker   PermissionChecker
}

// NewNamespaceStorage 创建项目 REST 存储。
func NewNamespaceStorage(nsStore NamespaceStore, wsStore WorkspaceStore, userStore UserStore, rbStore RoleBindingStore, checker PermissionChecker) rest.StandardStorage {
	return &namespaceStorage{nsStore: nsStore, wsStore: wsStore, userStore: userStore, rbStore: rbStore, checker: checker}
}

func (s *namespaceStorage) NewObject() runtime.Object { return &Namespace{} }

// +openapi:summary=获取项目详情
// +openapi:summary.workspaces.namespaces=获取工作空间下的项目详情
func (s *namespaceStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["namespaceId"]
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	ns, err := s.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	return namespaceWithOwnerToAPI(ns), nil
}

// +openapi:summary=获取项目列表
// +openapi:summary.workspaces.namespaces=获取工作空间下的项目列表
func (s *namespaceStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}

	// If called via /workspaces/{workspaceId}/namespaces, filter by workspace
	if wsID, ok := options.PathParams["workspaceId"]; ok && wsID != "" {
		query.Filters["workspace_id"] = wsID
	}

	// Inject access filter for non-admin users
	if af := filters.AccessFilterFromContext(ctx); af != nil && af.NamespaceIDs != nil {
		query.Filters["accessible_ids"] = af.NamespaceIDs
	}

	result, err := s.nsStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Namespace, len(result.Items))
	for i, item := range result.Items {
		items[i] = *namespaceWithOwnerToAPI(&item)
	}

	return &NamespaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "NamespaceList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=创建项目
// +openapi:summary.workspaces.namespaces=在工作空间下创建项目
func (s *namespaceStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	// If workspace ID comes from path params, use it
	if wsID, ok := options.PathParams["workspaceId"]; ok && wsID != "" {
		ns.Spec.WorkspaceID = wsID
	}

	// Auto-inject ownerId from authenticated user
	if userID, ok := oidc.UserIDFromContext(ctx); ok && ns.Spec.OwnerID == "" {
		ns.Spec.OwnerID = strconv.FormatInt(userID, 10)
	}

	if errs := ValidateNamespaceCreate(ns.ObjectMeta.Name, &ns.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ns, nil
	}

	workspaceID, err := parseID(ns.Spec.WorkspaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", ns.Spec.WorkspaceID), nil)
	}

	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	// Check workspace exists
	if _, err := s.wsStore.GetByID(ctx, workspaceID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("workspace %d not found", workspaceID), nil)
	}

	// Check owner exists
	if _, err := s.userStore.GetByID(ctx, ownerID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("owner user %d not found", ownerID), nil)
	}

	created, err := s.nsStore.Create(ctx, &DBNamespace{
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return namespaceWithOwnerToAPI(created), nil
}

// +openapi:summary=更新项目信息（全量）
// +openapi:summary.workspaces.namespaces=更新工作空间下的项目信息（全量）
func (s *namespaceStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	if errs := ValidateNamespaceUpdate(&ns.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ns, nil
	}

	id := options.PathParams["namespaceId"]
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	// Get existing to preserve workspace_id
	existingFull, err := s.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	workspaceID := existingFull.WorkspaceID
	if ns.Spec.WorkspaceID != "" {
		workspaceID, err = parseID(ns.Spec.WorkspaceID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", ns.Spec.WorkspaceID), nil)
		}
	}

	updated, err := s.nsStore.Update(ctx, &DBNamespace{
		ID:          nid,
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return namespaceToAPI(updated), nil
}

// +openapi:summary=更新项目信息（部分）
// +openapi:summary.workspaces.namespaces=更新工作空间下的项目信息（部分）
func (s *namespaceStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	id := options.PathParams["namespaceId"]

	if options.DryRun {
		existing, err := s.Get(ctx, &rest.GetOptions{PathParams: options.PathParams})
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	var workspaceID int64
	if ns.Spec.WorkspaceID != "" {
		workspaceID, err = parseID(ns.Spec.WorkspaceID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", ns.Spec.WorkspaceID), nil)
		}
	}

	var ownerID int64
	if ns.Spec.OwnerID != "" {
		ownerID, err = parseID(ns.Spec.OwnerID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
		}
	}

	patched, err := s.nsStore.Patch(ctx, nid, &DBNamespace{
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, err
	}

	return namespaceToAPI(patched), nil
}

// +openapi:summary=删除项目
// +openapi:summary.workspaces.namespaces=删除工作空间下的项目
func (s *namespaceStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id := options.PathParams["namespaceId"]
	nid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	// Collect affected user IDs before deletion (CASCADE will remove role_bindings)
	var affectedUserIDs []int64
	if s.rbStore != nil {
		affectedUserIDs, _ = s.rbStore.GetUserIDsByNamespaceID(ctx, nid)
	}

	if err := s.nsStore.Delete(ctx, nid); err != nil {
		return err
	}

	// Invalidate permission cache for affected users
	if s.checker != nil {
		for _, uid := range affectedUserIDs {
			s.checker.InvalidateCache(uid)
		}
	}
	return nil
}

// +openapi:summary=批量删除项目
// +openapi:summary.workspaces.namespaces=批量删除工作空间下的项目
func (s *namespaceStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		nid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, nid)
	}

	count, err := s.nsStore.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== workspaceUserStorage 工作空间成员存储 =====

// workspaceUserStorage 管理工作空间的成员关系，支持查询成员列表、批量添加和批量移除成员。
type workspaceUserStorage struct {
	rbStore   RoleBindingStore
	userStore UserStore
}

// NewWorkspaceUserStorage 创建工作空间成员管理 REST 存储。
func NewWorkspaceUserStorage(rbStore RoleBindingStore, userStore UserStore) rest.Storage {
	return &workspaceUserStorage{rbStore: rbStore, userStore: userStore}
}

func (s *workspaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

// +openapi:summary=获取工作空间成员详情
func (s *workspaceUserStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["userId"]
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	user, err := s.userStore.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return userToAPI(user), nil
}

// +openapi:summary=获取工作空间成员列表
func (s *workspaceUserStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.rbStore.ListWorkspaceMembers(ctx, wsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(result.Items))
	for i, m := range result.Items {
		u := userToAPI(&m.User)
		u.Spec.Role = m.Role
		u.Spec.JoinedAt = m.JoinedAt.Format(time.RFC3339)
		items[i] = *u
	}

	return &UserList{
		TypeMeta:   runtime.TypeMeta{Kind: "UserList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=批量添加工作空间成员
func (s *workspaceUserStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	req, ok := obj.(*BatchRequest)
	if !ok {
		return nil, fmt.Errorf("expected *BatchRequest, got %T", obj)
	}

	added, err := batchAddUsers(ctx, req.IDs, s.userStore, func(ctx context.Context, uid int64) (bool, error) {
		if err := s.rbStore.AddWorkspaceMember(ctx, uid, wsID); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		TypeMeta:     runtime.TypeMeta{Kind: "Result"},
		SuccessCount: added,
	}, nil
}

// +openapi:summary=批量移除工作空间成员
func (s *workspaceUserStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	count, err := batchRemoveUsers(ctx, ids, func(ctx context.Context, uid int64) error {
		return s.rbStore.RemoveWorkspaceMember(ctx, uid, wsID)
	})
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: count,
	}, nil
}

// ===== namespaceUserStorage 项目成员存储 =====

// namespaceUserStorage 管理项目的成员关系，支持查询成员列表、批量添加和批量移除成员。
// 添加项目成员时会自动将其加入父工作空间。
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/users
type namespaceUserStorage struct {
	rbStore   RoleBindingStore
	nsStore   NamespaceStore
	userStore UserStore
}

// NewNamespaceUserStorage 创建项目成员管理 REST 存储。
func NewNamespaceUserStorage(rbStore RoleBindingStore, nsStore NamespaceStore, userStore UserStore) rest.Storage {
	return &namespaceUserStorage{rbStore: rbStore, nsStore: nsStore, userStore: userStore}
}

func (s *namespaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

// +openapi:summary=获取项目成员详情
// +openapi:summary.workspaces.namespaces.users=获取工作空间下项目的成员详情
func (s *namespaceUserStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["userId"]
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	user, err := s.userStore.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return userToAPI(user), nil
}

// +openapi:summary=获取项目成员列表
// +openapi:summary.workspaces.namespaces.users=获取工作空间下项目的成员列表
func (s *namespaceUserStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.rbStore.ListNamespaceMembers(ctx, nsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(result.Items))
	for i, m := range result.Items {
		u := userToAPI(&m.User)
		u.Spec.Role = m.Role
		u.Spec.JoinedAt = m.JoinedAt.Format(time.RFC3339)
		items[i] = *u
	}

	return &UserList{
		TypeMeta:   runtime.TypeMeta{Kind: "UserList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=批量添加项目成员
// +openapi:summary.workspaces.namespaces.users=批量添加工作空间下项目的成员
func (s *namespaceUserStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	req, ok := obj.(*BatchRequest)
	if !ok {
		return nil, fmt.Errorf("expected *BatchRequest, got %T", obj)
	}

	// Check max members limit
	ns, err := s.nsStore.GetByID(ctx, nsID)
	if err != nil {
		return nil, err
	}
	if ns.MaxMembers > 0 {
		currentCount, err := s.nsStore.CountUsers(ctx, nsID)
		if err != nil {
			return nil, err
		}
		if currentCount+int64(len(req.IDs)) > int64(ns.MaxMembers) {
			return nil, apierrors.NewBadRequest(
				fmt.Sprintf("namespace member limit exceeded: current %d, adding %d, max %d", currentCount, len(req.IDs), ns.MaxMembers),
				nil,
			)
		}
	}

	added, err := batchAddUsers(ctx, req.IDs, s.userStore, func(ctx context.Context, uid int64) (bool, error) {
		if err := s.rbStore.AddNamespaceMember(ctx, uid, nsID); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		TypeMeta:     runtime.TypeMeta{Kind: "Result"},
		SuccessCount: added,
	}, nil
}

// +openapi:summary=批量移除项目成员
// +openapi:summary.workspaces.namespaces.users=批量移除工作空间下项目的成员
func (s *namespaceUserStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	count, err := batchRemoveUsers(ctx, ids, func(ctx context.Context, uid int64) error {
		return s.rbStore.RemoveNamespaceMember(ctx, uid, nsID)
	})
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: count,
	}, nil
}

// ===== userWorkspaceVerbStorage 用户工作空间视图 =====

// userWorkspaceVerbStorage 用户关联工作空间的 custom verb 存储，支持分页、筛选和排序。
// 注册为 GET /users/{userId}:workspaces
type userWorkspaceVerbStorage struct {
	rbStore RoleBindingStore
}

// NewUserWorkspacesVerb 创建用户工作空间视图存储。
// +openapi:customverb=workspaces
// +openapi:resource=User
// +openapi:summary=获取用户关联的工作空间列表
func NewUserWorkspacesVerb(rbStore RoleBindingStore) rest.Lister {
	return &userWorkspaceVerbStorage{rbStore: rbStore}
}

func (s *userWorkspaceVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	uid, err := parseID(options.PathParams["userId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", options.PathParams["userId"]), nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.rbStore.ListUserWorkspaces(ctx, uid, query)
	if err != nil {
		return nil, err
	}

	items := make([]Workspace, len(result.Items))
	for i, item := range result.Items {
		ws := workspaceWithOwnerToAPI(&DBWorkspaceWithOwner{
			Workspace:      item.Workspace,
			OwnerUsername:  item.OwnerUsername,
			NamespaceCount: item.NamespaceCount,
			MemberCount:    item.MemberCount,
		})
		ws.Spec.Role = item.Role
		ws.Spec.RoleDisplayName = item.RoleDisplayName
		ws.Spec.JoinedAt = item.JoinedAt.Format(time.RFC3339)
		items[i] = *ws
	}

	return &WorkspaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "WorkspaceList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== userNamespaceVerbStorage 用户项目视图 =====

// userNamespaceVerbStorage 用户关联项目的 custom verb 存储，支持分页、筛选和排序。
// 注册为 GET /users/{userId}:namespaces
type userNamespaceVerbStorage struct {
	rbStore RoleBindingStore
}

// NewUserNamespacesVerb 创建用户项目视图存储。
// +openapi:customverb=namespaces
// +openapi:resource=User
// +openapi:summary=获取用户关联的项目列表
func NewUserNamespacesVerb(rbStore RoleBindingStore) rest.Lister {
	return &userNamespaceVerbStorage{rbStore: rbStore}
}

func (s *userNamespaceVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	uid, err := parseID(options.PathParams["userId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", options.PathParams["userId"]), nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.rbStore.ListUserNamespaces(ctx, uid, query)
	if err != nil {
		return nil, err
	}

	items := make([]Namespace, len(result.Items))
	for i, item := range result.Items {
		ns := namespaceWithOwnerToAPI(&DBNamespaceWithOwner{
			Namespace:     item.Namespace,
			OwnerUsername: item.OwnerUsername,
			WorkspaceName: item.WorkspaceName,
			MemberCount:   item.MemberCount,
		})
		ns.Spec.Role = item.Role
		ns.Spec.RoleDisplayName = item.RoleDisplayName
		ns.Spec.JoinedAt = item.JoinedAt.Format(time.RFC3339)
		items[i] = *ns
	}

	return &NamespaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "NamespaceList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// ===== userRoleBindingsVerbStorage 用户角色绑定视图 =====

// userRoleBindingsVerbStorage 用户角色绑定的 custom verb 存储。
// 注册为 GET /users/{userId}:rolebindings
type userRoleBindingsVerbStorage struct {
	rbStore RoleBindingStore
}

// NewUserRoleBindingsVerb 创建用户角色绑定视图存储。
// +openapi:customverb=rolebindings
// +openapi:resource=User
// +openapi:response=RoleBindingList
// +openapi:summary=获取用户的角色绑定列表
func NewUserRoleBindingsVerb(rbStore RoleBindingStore) rest.Lister {
	return &userRoleBindingsVerbStorage{rbStore: rbStore}
}

func (s *userRoleBindingsVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	uid, err := parseID(options.PathParams["userId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", options.PathParams["userId"]), nil)
	}

	query := restOptionsToListQuery(options)

	result, err := s.rbStore.ListByUserID(ctx, uid, query)
	if err != nil {
		return nil, err
	}

	return roleBindingListToAPI(result), nil
}

// ===== userPermissionsVerbStorage 用户权限视图 =====

// userPermissionsVerbStorage 用户权限聚合视图的 custom verb 存储。
// 注册为 GET /users/{userId}:permissions
type userPermissionsVerbStorage struct {
	rbStore   RoleBindingStore
	permStore PermissionStore
}

// NewUserPermissionsVerb 创建用户权限视图存储。
// +openapi:customverb=permissions
// +openapi:resource=User
// +openapi:response=UserPermissions
// +openapi:summary=获取用户的权限视图
func NewUserPermissionsVerb(rbStore RoleBindingStore, permStore PermissionStore) rest.Lister {
	return &userPermissionsVerbStorage{rbStore: rbStore, permStore: permStore}
}

func (s *userPermissionsVerbStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	uid, err := parseID(options.PathParams["userId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", options.PathParams["userId"]), nil)
	}

	// 1. Get all role bindings with rules for this user
	rows, err := s.rbStore.GetUserRoleBindingsWithRules(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 2. Get all registered permission codes for pattern expansion
	allCodes, err := s.permStore.ListAllCodes(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Group by scope and collect patterns + role names
	var platformPatterns []string
	platformRoleNames := make(map[string]bool)
	isPlatformAdmin := false

	type wsEntry struct {
		patterns  []string
		roleNames map[string]bool
	}
	type nsEntry struct {
		patterns    []string
		roleNames   map[string]bool
		workspaceID string
	}
	wsMap := make(map[string]*wsEntry)
	nsMap := make(map[string]*nsEntry)

	for _, row := range rows {
		switch row.Scope {
		case ScopePlatform:
			platformPatterns = append(platformPatterns, row.Pattern)
			platformRoleNames[row.RoleName] = true
			if row.Pattern == "*:*" {
				isPlatformAdmin = true
			}
		case ScopeWorkspace:
			if row.WorkspaceID != nil {
				wsIDStr := strconv.FormatInt(*row.WorkspaceID, 10)
				entry, ok := wsMap[wsIDStr]
				if !ok {
					entry = &wsEntry{roleNames: make(map[string]bool)}
					wsMap[wsIDStr] = entry
				}
				entry.patterns = append(entry.patterns, row.Pattern)
				entry.roleNames[row.RoleName] = true
			}
		case ScopeNamespace:
			if row.NamespaceID != nil {
				nsIDStr := strconv.FormatInt(*row.NamespaceID, 10)
				entry, ok := nsMap[nsIDStr]
				if !ok {
					var wsIDStr string
					if row.WorkspaceID != nil {
						wsIDStr = strconv.FormatInt(*row.WorkspaceID, 10)
					}
					entry = &nsEntry{roleNames: make(map[string]bool), workspaceID: wsIDStr}
					nsMap[nsIDStr] = entry
				}
				entry.patterns = append(entry.patterns, row.Pattern)
				entry.roleNames[row.RoleName] = true
			}
		}
	}

	// 4. Expand patterns to concrete permission codes
	spec := UserPermissionsSpec{
		IsPlatformAdmin: isPlatformAdmin,
		Platform:        ExpandPatterns(platformPatterns, allCodes),
		Workspaces:      make(map[string]WorkspaceScopePerms),
		Namespaces:      make(map[string]NamespaceScopePerms),
	}

	for wsID, entry := range wsMap {
		spec.Workspaces[wsID] = WorkspaceScopePerms{
			RoleNames:   mapKeys(entry.roleNames),
			Permissions: ExpandPatterns(entry.patterns, allCodes),
		}
	}

	for nsID, entry := range nsMap {
		spec.Namespaces[nsID] = NamespaceScopePerms{
			RoleNames:   mapKeys(entry.roleNames),
			WorkspaceID: entry.workspaceID,
			Permissions: ExpandPatterns(entry.patterns, allCodes),
		}
	}

	return &UserPermissions{
		TypeMeta: runtime.TypeMeta{Kind: "UserPermissions"},
		Spec:     spec,
	}, nil
}

// mapKeys returns the keys of a map as a sorted slice.
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// restOptionsToListQuery converts REST ListOptions to a db.ListQuery.
func restOptionsToListQuery(options *rest.ListOptions) db.ListQuery {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}
	return query
}

// ===== change-password 修改密码操作 =====

// ChangePasswordRequest 修改密码请求：包含旧密码和新密码。
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// StatusResponse 操作结果响应。
type StatusResponse struct {
	runtime.TypeMeta `json:",inline"`
	Status           string `json:"status"`
	Message          string `json:"message"`
}

func (s *StatusResponse) GetTypeMeta() *runtime.TypeMeta { return &s.TypeMeta }

// NewChangePasswordHandler 创建修改密码的操作处理器。验证旧密码后设置新密码，并吊销该用户所有已有的刷新令牌。
// +openapi:action=change-password
// +openapi:resource=User
// +openapi:summary=修改用户密码
func NewChangePasswordHandler(userStore UserStore, refreshStore RefreshTokenStore, hashPasswd PasswordHasher, verifyPasswd func(password, hash string) error) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		idStr := params["userId"]
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}

		var req ChangePasswordRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}

		if req.OldPassword == "" || req.NewPassword == "" {
			return nil, apierrors.NewBadRequest("oldPassword and newPassword are required", nil)
		}

		if errs := ValidatePassword(req.NewPassword); errs.HasErrors() {
			return nil, apierrors.NewBadRequest("validation failed", errs)
		}

		// Get user first to find their username, then get auth data
		user, err := userStore.GetByID(ctx, uid)
		if err != nil {
			return nil, err
		}
		authUser, err := userStore.GetUserForAuth(ctx, user.Username)
		if err != nil {
			return nil, err
		}

		// Verify old password
		if err := verifyPasswd(req.OldPassword, authUser.PasswordHash); err != nil {
			return nil, apierrors.NewBadRequest("old password is incorrect", nil)
		}

		// Hash and set new password
		hash, err := hashPasswd(req.NewPassword)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("hash password: %w", err))
		}
		if err := userStore.SetPasswordHash(ctx, uid, hash); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("set password: %w", err))
		}

		// Revoke all existing refresh tokens for this user
		if refreshStore != nil {
			if err := refreshStore.RevokeByUserID(ctx, uid); err != nil {
				logger.Infof("failed to revoke refresh tokens for user %d: %v", uid, err)
			}
		}

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "password changed successfully",
		}, nil
	}
}

// ===== transfer-ownership 转移所有权操作 =====

// TransferOwnershipRequest 转移所有权请求。
type TransferOwnershipRequest struct {
	NewOwnerUserID string `json:"newOwnerUserId"`
}

// NewTransferOwnershipHandler 创建工作空间所有权转移处理器。仅当前 owner 或 platform-admin 可操作。
// +openapi:action=transfer-ownership
// +openapi:resource=Workspace
// +openapi:summary=转移工作空间所有权
func NewTransferOwnershipHandler(rbStore RoleBindingStore, checker *RBACChecker) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		resourceID, err := parseID(params["workspaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", params["workspaceId"]), nil)
		}

		var req TransferOwnershipRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}
		if req.NewOwnerUserID == "" {
			return nil, apierrors.NewBadRequest("newOwnerUserId is required", nil)
		}
		newOwnerUID, err := parseID(req.NewOwnerUserID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid newOwnerUserId: %s", req.NewOwnerUserID), nil)
		}

		callerID, ok := oidc.UserIDFromContext(ctx)
		if !ok {
			return nil, apierrors.NewForbidden("authentication required")
		}

		isPlatformAdmin, err := checker.IsPlatformAdmin(ctx, callerID)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("check platform admin: %w", err))
		}

		oldOwnerUID, err := rbStore.TransferOwnership(ctx, ScopeWorkspace, resourceID, callerID, isPlatformAdmin, newOwnerUID, RoleWorkspaceAdmin)
		if err != nil {
			return nil, err
		}

		sharedPermCache.Invalidate(oldOwnerUID)
		sharedPermCache.Invalidate(newOwnerUID)

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "workspace ownership transferred successfully",
		}, nil
	}
}

// NewNamespaceTransferOwnershipHandler 创建项目所有权转移处理器。仅当前 owner 或 platform-admin 可操作。
// +openapi:action=transfer-ownership
// +openapi:resource=Namespace
// +openapi:summary=转移项目所有权
func NewNamespaceTransferOwnershipHandler(rbStore RoleBindingStore, checker *RBACChecker) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		nsID, err := parseID(params["namespaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", params["namespaceId"]), nil)
		}

		var req TransferOwnershipRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, apierrors.NewBadRequest("invalid request body", nil)
		}
		if req.NewOwnerUserID == "" {
			return nil, apierrors.NewBadRequest("newOwnerUserId is required", nil)
		}
		newOwnerUID, err := parseID(req.NewOwnerUserID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid newOwnerUserId: %s", req.NewOwnerUserID), nil)
		}

		callerID, ok := oidc.UserIDFromContext(ctx)
		if !ok {
			return nil, apierrors.NewForbidden("authentication required")
		}

		isPlatformAdmin, err := checker.IsPlatformAdmin(ctx, callerID)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("check platform admin: %w", err))
		}

		oldOwnerUID, err := rbStore.TransferOwnership(ctx, ScopeNamespace, nsID, callerID, isPlatformAdmin, newOwnerUID, RoleNamespaceAdmin)
		if err != nil {
			return nil, err
		}

		sharedPermCache.Invalidate(oldOwnerUID)
		sharedPermCache.Invalidate(newOwnerUID)

		return &StatusResponse{
			TypeMeta: runtime.TypeMeta{Kind: "Status"},
			Status:   "Success",
			Message:  "namespace ownership transferred successfully",
		}, nil
	}
}

// ===== helpers =====

func userToAPI(u *DBUser) *User {
	return &User{
		TypeMeta: runtime.TypeMeta{Kind: "User"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(u.ID, 10),
			Name:      u.Username,
			CreatedAt: new(u.CreatedAt),
			UpdatedAt: new(u.UpdatedAt),
		},
		Spec: UserSpec{
			Username:    u.Username,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Phone:       u.Phone,
			AvatarURL:   u.AvatarUrl,
			Status:      u.Status,
		},
	}
}

func userWithNamespacesToAPI(u *DBUserWithNamespaces) *User {
	user := userToAPI(&u.User)
	if len(u.NamespaceNames) > 0 {
		user.Spec.Namespaces = u.NamespaceNames
	}
	return user
}

func workspaceToAPI(w *DBWorkspace) *Workspace {
	return &Workspace{
		TypeMeta: runtime.TypeMeta{Kind: "Workspace"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(w.ID, 10),
			Name:      w.Name,
			CreatedAt: new(w.CreatedAt),
			UpdatedAt: new(w.UpdatedAt),
		},
		Spec: WorkspaceSpec{
			DisplayName: w.DisplayName,
			Description: w.Description,
			OwnerID:     strconv.FormatInt(w.OwnerID, 10),
			Status:      w.Status,
		},
	}
}

func workspaceWithOwnerToAPI(w *DBWorkspaceWithOwner) *Workspace {
	ws := workspaceToAPI(&w.Workspace)
	ws.Spec.OwnerName = w.OwnerUsername
	ws.Spec.NamespaceCount = int(w.NamespaceCount)
	ws.Spec.MemberCount = int(w.MemberCount)
	return ws
}

func namespaceWithOwnerToAPI(n *DBNamespaceWithOwner) *Namespace {
	ns := namespaceToAPI(&n.Namespace)
	ns.Spec.OwnerName = n.OwnerUsername
	ns.Spec.WorkspaceName = n.WorkspaceName
	ns.Spec.MemberCount = int(n.MemberCount)
	return ns
}

func namespaceToAPI(n *DBNamespace) *Namespace {
	return &Namespace{
		TypeMeta: runtime.TypeMeta{Kind: "Namespace"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: new(n.CreatedAt),
			UpdatedAt: new(n.UpdatedAt),
		},
		Spec: NamespaceSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			WorkspaceID: strconv.FormatInt(n.WorkspaceID, 10),
			OwnerID:     strconv.FormatInt(n.OwnerID, 10),
			Visibility:  n.Visibility,
			MaxMembers:  int(n.MaxMembers),
			Status:      n.Status,
		},
	}
}

// ensureUserExists verifies a user exists by ID, returning a BadRequest error if not.
func ensureUserExists(ctx context.Context, store UserStore, id int64) error {
	if _, err := store.GetByID(ctx, id); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("user %d not found", id), nil)
	}
	return nil
}

// batchAddUsers validates and adds users via the provided addFn.
// Returns the count of successfully added users.
func batchAddUsers(ctx context.Context, ids []string, userStore UserStore, addFn func(ctx context.Context, uid int64) (bool, error)) (int, error) {
	added := 0
	for _, idStr := range ids {
		uid, err := parseID(idStr)
		if err != nil {
			return 0, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		if err := ensureUserExists(ctx, userStore, uid); err != nil {
			return 0, err
		}
		ok, err := addFn(ctx, uid)
		if err != nil {
			return 0, err
		}
		if ok {
			added++
		}
	}
	return added, nil
}

// batchRemoveUsers removes users via the provided removeFn.
func batchRemoveUsers(ctx context.Context, ids []string, removeFn func(ctx context.Context, uid int64) error) (int, error) {
	for _, idStr := range ids {
		uid, err := parseID(idStr)
		if err != nil {
			return 0, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		if err := removeFn(ctx, uid); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// --- Role Storage (CRUD) ---

type roleStorage struct {
	roleStore RoleStore
	rbStore   RoleBindingStore
}

// NewRoleStorage creates a role REST storage with full CRUD.
func NewRoleStorage(roleStore RoleStore, rbStore RoleBindingStore) rest.Storage {
	return &roleStorage{roleStore: roleStore, rbStore: rbStore}
}

func (s *roleStorage) NewObject() runtime.Object { return &Role{} }

// +openapi:summary=获取平台角色详情
func (s *roleStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	role, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	if role.Scope != ScopePlatform {
		return nil, apierrors.NewNotFound("role", id)
	}

	return roleWithRulesToAPI(role), nil
}

// +openapi:summary=获取平台角色列表
func (s *roleStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	// Force platform scope — only return platform roles
	query.Filters["scope"] = ScopePlatform
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}

	result, err := s.roleStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Role, len(result.Items))
	for i, item := range result.Items {
		items[i] = *roleListRowToAPI(&item)
	}

	return &RoleList{
		TypeMeta:   runtime.TypeMeta{Kind: "RoleList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=创建平台角色
func (s *roleStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}

	// Force platform scope — scoped roles must be created via workspace/namespace endpoints
	role.Spec.Scope = ScopePlatform

	if errs := ValidateRoleCreate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return role, nil
	}

	dbRole := &DBRole{
		Name:        role.Spec.Name,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
		Scope:       ScopePlatform,
	}

	created, err := s.roleStore.Create(ctx, dbRole)
	if err != nil {
		return nil, err
	}

	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, created.ID, role.Spec.Rules); err != nil {
			return nil, err
		}
	}

	// Re-fetch to include rules
	withRules, err := s.roleStore.GetByID(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return roleWithRulesToAPI(withRules), nil
}

// +openapi:summary=更新平台角色
func (s *roleStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}

	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if existing.Scope != ScopePlatform {
		return nil, apierrors.NewNotFound("role", id)
	}
	if existing.Builtin {
		return nil, apierrors.NewBadRequest("cannot modify built-in role", nil)
	}

	if errs := ValidateRoleUpdate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return role, nil
	}

	dbRole := &DBRole{
		ID:          rid,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
	}

	if _, err := s.roleStore.Update(ctx, dbRole); err != nil {
		return nil, err
	}

	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, rid, role.Spec.Rules); err != nil {
			return nil, err
		}
	}

	withRules, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	return roleWithRulesToAPI(withRules), nil
}

// +openapi:summary=删除平台角色
func (s *roleStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return err
	}
	if existing.Scope != ScopePlatform {
		return apierrors.NewNotFound("role", id)
	}
	if existing.Builtin {
		return apierrors.NewBadRequest("cannot delete built-in role", nil)
	}

	count, err := s.rbStore.CountByRoleAndScope(ctx, rid, ScopePlatform)
	if err != nil {
		return err
	}
	if count > 0 {
		return apierrors.NewBadRequest("cannot delete role with active bindings", nil)
	}

	if options.DryRun {
		return nil
	}

	return s.roleStore.Delete(ctx, rid)
}

func roleToAPI(r *DBRole) *Role {
	return &Role{
		TypeMeta: runtime.TypeMeta{Kind: "Role"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			CreatedAt: &r.CreatedAt,
			UpdatedAt: &r.UpdatedAt,
		},
		Spec: RoleSpec{
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Description: r.Description,
			Scope:       r.Scope,
			Builtin:     r.Builtin,
		},
	}
}

func roleListRowToAPI(r *DBRoleListRow) *Role {
	rc := r.RuleCount
	return &Role{
		TypeMeta: runtime.TypeMeta{Kind: "Role"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(r.ID, 10),
			CreatedAt: &r.CreatedAt,
			UpdatedAt: &r.UpdatedAt,
		},
		Spec: RoleSpec{
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Description: r.Description,
			Scope:       r.Scope,
			Builtin:     r.Builtin,
			RuleCount:   &rc,
		},
	}
}

func roleWithRulesToAPI(r *DBRoleWithRules) *Role {
	role := roleToAPI(&r.Role)
	role.Spec.Rules = r.Rules
	return role
}

// --- Scoped Role Storage (full CRUD, for workspace/namespace sub-resource) ---

// scopedRoleStorage 作用域角色的完整 CRUD 存储，按 scope 过滤角色列表。
// 注册为 /workspaces/{workspaceId}/roles、/namespaces/{namespaceId}/roles
// 和 /workspaces/{workspaceId}/namespaces/{namespaceId}/roles。
// +openapi:resource=Role
// +openapi:path=/workspaces/{workspaceId}/roles
// +openapi:path=/namespaces/{namespaceId}/roles
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/roles
type scopedRoleStorage struct {
	roleStore RoleStore
	rbStore   RoleBindingStore
	scope     string // ScopeWorkspace or ScopeNamespace
}

// NewScopedRoleStorage creates a scoped role storage with full CRUD.
func NewScopedRoleStorage(roleStore RoleStore, rbStore RoleBindingStore, scope string) rest.Storage {
	return &scopedRoleStorage{roleStore: roleStore, rbStore: rbStore, scope: scope}
}

func (s *scopedRoleStorage) NewObject() runtime.Object { return &Role{} }

// +openapi:summary=获取工作空间角色列表
// +openapi:summary.namespaces.roles=获取项目角色列表
// +openapi:summary.workspaces.namespaces.roles=获取工作空间下项目的角色列表
func (s *scopedRoleStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)
	query.Filters["scope"] = s.scope

	if s.scope == ScopeWorkspace {
		wsID, err := parseID(options.PathParams["workspaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
		}
		query.Filters["workspace_id"] = wsID
	} else {
		nsID, err := parseID(options.PathParams["namespaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
		}
		query.Filters["namespace_id"] = nsID
	}

	result, err := s.roleStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Role, len(result.Items))
	for i, item := range result.Items {
		items[i] = *roleListRowToAPI(&item)
	}

	return &RoleList{
		TypeMeta:   runtime.TypeMeta{Kind: "RoleList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=获取工作空间角色详情
// +openapi:summary.namespaces.roles=获取项目角色详情
// +openapi:summary.workspaces.namespaces.roles=获取工作空间下项目的角色详情
func (s *scopedRoleStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	role, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	if role.Scope != s.scope {
		return nil, apierrors.NewNotFound("role", id)
	}
	if s.scope == ScopeWorkspace {
		wsID, _ := parseID(options.PathParams["workspaceId"])
		if role.WorkspaceID == nil || *role.WorkspaceID != wsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	} else {
		nsID, _ := parseID(options.PathParams["namespaceId"])
		if role.NamespaceID == nil || *role.NamespaceID != nsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	}

	return roleWithRulesToAPI(role), nil
}

// +openapi:summary=创建工作空间自定义角色
// +openapi:summary.namespaces.roles=创建项目自定义角色
// +openapi:summary.workspaces.namespaces.roles=创建工作空间下项目的自定义角色
func (s *scopedRoleStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}

	role.Spec.Scope = s.scope

	if errs := ValidateRoleCreate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return role, nil
	}

	dbRole := &DBRole{
		Name:        role.Spec.Name,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
		Scope:       s.scope,
	}

	if s.scope == ScopeWorkspace {
		wsID, err := parseID(options.PathParams["workspaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
		}
		dbRole.WorkspaceID = &wsID
	} else {
		nsID, err := parseID(options.PathParams["namespaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
		}
		dbRole.NamespaceID = &nsID
	}

	created, err := s.roleStore.Create(ctx, dbRole)
	if err != nil {
		return nil, err
	}

	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, created.ID, role.Spec.Rules); err != nil {
			return nil, err
		}
	}

	withRules, err := s.roleStore.GetByID(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return roleWithRulesToAPI(withRules), nil
}

// +openapi:summary=更新工作空间角色
// +openapi:summary.namespaces.roles=更新项目角色
// +openapi:summary.workspaces.namespaces.roles=更新工作空间下项目的角色
func (s *scopedRoleStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}

	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if existing.Builtin {
		return nil, apierrors.NewBadRequest("cannot modify built-in role", nil)
	}

	// Verify ownership
	if existing.Scope != s.scope {
		return nil, apierrors.NewNotFound("role", id)
	}
	if s.scope == ScopeWorkspace {
		wsID, _ := parseID(options.PathParams["workspaceId"])
		if existing.WorkspaceID == nil || *existing.WorkspaceID != wsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	} else {
		nsID, _ := parseID(options.PathParams["namespaceId"])
		if existing.NamespaceID == nil || *existing.NamespaceID != nsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	}

	if errs := ValidateRoleUpdate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return role, nil
	}

	dbRole := &DBRole{
		ID:          rid,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
	}

	if _, err := s.roleStore.Update(ctx, dbRole); err != nil {
		return nil, err
	}

	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, rid, role.Spec.Rules); err != nil {
			return nil, err
		}
	}

	withRules, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}

	return roleWithRulesToAPI(withRules), nil
}

// +openapi:summary=删除工作空间角色
// +openapi:summary.namespaces.roles=删除项目角色
// +openapi:summary.workspaces.namespaces.roles=删除工作空间下项目的角色
func (s *scopedRoleStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}

	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return apierrors.NewBadRequest("cannot delete built-in role", nil)
	}

	// Verify ownership
	if existing.Scope != s.scope {
		return apierrors.NewNotFound("role", id)
	}
	if s.scope == ScopeWorkspace {
		wsID, _ := parseID(options.PathParams["workspaceId"])
		if existing.WorkspaceID == nil || *existing.WorkspaceID != wsID {
			return apierrors.NewNotFound("role", id)
		}
	} else {
		nsID, _ := parseID(options.PathParams["namespaceId"])
		if existing.NamespaceID == nil || *existing.NamespaceID != nsID {
			return apierrors.NewNotFound("role", id)
		}
	}

	// Check no bindings exist
	count, err := s.rbStore.CountByRoleAndScope(ctx, rid, s.scope)
	if err != nil {
		return err
	}
	if count > 0 {
		return apierrors.NewBadRequest("cannot delete role with active bindings", nil)
	}

	if options.DryRun {
		return nil
	}

	return s.roleStore.Delete(ctx, rid)
}

// --- Permission Storage (read-only) ---

type permissionStorage struct {
	permStore PermissionStore
}

// NewPermissionStorage creates a read-only permission REST storage.
func NewPermissionStorage(permStore PermissionStore) rest.Storage {
	return &permissionStorage{permStore: permStore}
}

func (s *permissionStorage) NewObject() runtime.Object { return &Permission{} }

// +openapi:summary=获取权限列表
func (s *permissionStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	if module := options.Filters["module"]; module != "" {
		query.Filters["module_prefix"] = module + ":"
	}
	if search := options.Filters["search"]; search != "" {
		query.Filters["search"] = search
	}
	if scope := options.Filters["scope"]; scope != "" {
		query.Filters["scope"] = scope
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}

	result, err := s.permStore.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Permission, len(result.Items))
	for i, item := range result.Items {
		items[i] = *permissionToAPI(&item)
	}

	return &PermissionList{
		TypeMeta:   runtime.TypeMeta{Kind: "PermissionList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// --- RoleBinding Storage (three-level) ---

// ===== roleBindingStorage 平台级角色绑定 =====

// +openapi:resource=RoleBinding
// +openapi:path=/rolebindings
type roleBindingStorage struct {
	rbStore   RoleBindingStore
	roleStore RoleStore
}

// NewRoleBindingStorage creates a platform-level role binding REST storage.
func NewRoleBindingStorage(rbStore RoleBindingStore, roleStore RoleStore) rest.Storage {
	return &roleBindingStorage{rbStore: rbStore, roleStore: roleStore}
}

func (s *roleBindingStorage) NewObject() runtime.Object { return &RoleBinding{} }

// +openapi:summary=获取平台级角色绑定列表
func (s *roleBindingStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)
	result, err := s.rbStore.ListPlatform(ctx, query)
	if err != nil {
		return nil, err
	}
	return roleBindingListToAPI(result), nil
}

// +openapi:summary=创建平台级角色绑定
func (s *roleBindingStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	rb, ok := obj.(*RoleBinding)
	if !ok {
		return nil, fmt.Errorf("expected *RoleBinding, got %T", obj)
	}

	if errs := ValidateRoleBindingCreate(&rb.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	roleID, err := parseID(rb.Spec.RoleID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", rb.Spec.RoleID), nil)
	}
	role, err := s.roleStore.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.Scope != ScopePlatform {
		return nil, apierrors.NewBadRequest("role scope must be 'platform' for platform-level bindings", nil)
	}

	userID, err := parseID(rb.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", rb.Spec.UserID), nil)
	}

	if options.DryRun {
		return rb, nil
	}

	created, err := s.rbStore.Create(ctx, &DBRoleBinding{
		UserID: userID,
		RoleID: roleID,
		Scope:  ScopePlatform,
	})
	if err != nil {
		return nil, err
	}

	sharedPermCache.Invalidate(userID)

	return roleBindingToAPI(created, "", "", role.Name, role.DisplayName), nil
}

// +openapi:summary=删除平台级角色绑定
func (s *roleBindingStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["rolebindingId"]
	rbID, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role binding ID: %s", id), nil)
	}

	existing, err := s.rbStore.GetByID(ctx, rbID)
	if err != nil {
		return err
	}
	if existing.IsOwner {
		return apierrors.NewBadRequest("cannot delete owner role binding", nil)
	}

	if options.DryRun {
		return nil
	}

	if err := s.rbStore.Delete(ctx, rbID); err != nil {
		return err
	}

	sharedPermCache.Invalidate(existing.UserID)

	return nil
}

// ===== workspaceRoleBindingStorage 工作空间级角色绑定 =====

// +openapi:resource=RoleBinding
// +openapi:path=/workspaces/{workspaceId}/rolebindings
type workspaceRoleBindingStorage struct {
	rbStore   RoleBindingStore
	roleStore RoleStore
}

// NewWorkspaceRoleBindingStorage creates a workspace-level role binding REST storage.
func NewWorkspaceRoleBindingStorage(rbStore RoleBindingStore, roleStore RoleStore) rest.Storage {
	return &workspaceRoleBindingStorage{rbStore: rbStore, roleStore: roleStore}
}

func (s *workspaceRoleBindingStorage) NewObject() runtime.Object { return &RoleBinding{} }

// +openapi:summary=获取工作空间级角色绑定列表
func (s *workspaceRoleBindingStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}
	query := restOptionsToListQuery(options)
	result, err := s.rbStore.ListByWorkspaceID(ctx, wsID, query)
	if err != nil {
		return nil, err
	}
	return roleBindingListToAPI(result), nil
}

// +openapi:summary=创建工作空间级角色绑定
func (s *workspaceRoleBindingStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	rb, ok := obj.(*RoleBinding)
	if !ok {
		return nil, fmt.Errorf("expected *RoleBinding, got %T", obj)
	}

	if errs := ValidateRoleBindingCreate(&rb.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	roleID, err := parseID(rb.Spec.RoleID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", rb.Spec.RoleID), nil)
	}
	role, err := s.roleStore.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.Scope != ScopeWorkspace {
		return nil, apierrors.NewBadRequest("role scope must be 'workspace' for workspace-level bindings", nil)
	}
	if role.WorkspaceID == nil || *role.WorkspaceID != wsID {
		return nil, apierrors.NewBadRequest("role does not belong to this workspace", nil)
	}

	userID, err := parseID(rb.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", rb.Spec.UserID), nil)
	}

	if options.DryRun {
		return rb, nil
	}

	created, err := s.rbStore.Create(ctx, &DBRoleBinding{
		UserID:      userID,
		RoleID:      roleID,
		Scope:       ScopeWorkspace,
		WorkspaceID: &wsID,
	})
	if err != nil {
		return nil, err
	}

	sharedPermCache.Invalidate(userID)

	return roleBindingToAPI(created, "", "", role.Name, role.DisplayName), nil
}

// +openapi:summary=删除工作空间级角色绑定
func (s *workspaceRoleBindingStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["rolebindingId"]
	rbID, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role binding ID: %s", id), nil)
	}

	existing, err := s.rbStore.GetByID(ctx, rbID)
	if err != nil {
		return err
	}
	if existing.IsOwner {
		return apierrors.NewBadRequest("cannot delete owner role binding", nil)
	}

	if options.DryRun {
		return nil
	}

	if err := s.rbStore.Delete(ctx, rbID); err != nil {
		return err
	}

	sharedPermCache.Invalidate(existing.UserID)

	return nil
}

// ===== namespaceRoleBindingStorage 项目级角色绑定 =====

// +openapi:resource=RoleBinding
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/rolebindings
type namespaceRoleBindingStorage struct {
	rbStore   RoleBindingStore
	roleStore RoleStore
	nsStore   NamespaceStore
}

// NewNamespaceRoleBindingStorage creates a namespace-level role binding REST storage.
func NewNamespaceRoleBindingStorage(rbStore RoleBindingStore, roleStore RoleStore, nsStore NamespaceStore) rest.Storage {
	return &namespaceRoleBindingStorage{rbStore: rbStore, roleStore: roleStore, nsStore: nsStore}
}

func (s *namespaceRoleBindingStorage) NewObject() runtime.Object { return &RoleBinding{} }

// +openapi:summary=获取项目级角色绑定列表
// +openapi:summary.workspaces.namespaces.rolebindings=获取工作空间下项目的角色绑定列表
func (s *namespaceRoleBindingStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}
	query := restOptionsToListQuery(options)
	result, err := s.rbStore.ListByNamespaceID(ctx, nsID, query)
	if err != nil {
		return nil, err
	}
	return roleBindingListToAPI(result), nil
}

// +openapi:summary=创建项目级角色绑定
// +openapi:summary.workspaces.namespaces.rolebindings=创建工作空间下项目的角色绑定
func (s *namespaceRoleBindingStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	rb, ok := obj.(*RoleBinding)
	if !ok {
		return nil, fmt.Errorf("expected *RoleBinding, got %T", obj)
	}

	if errs := ValidateRoleBindingCreate(&rb.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	// Look up namespace to get workspace ID
	ns, err := s.nsStore.GetByID(ctx, nsID)
	if err != nil {
		return nil, err
	}
	wsID := ns.WorkspaceID

	roleID, err := parseID(rb.Spec.RoleID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", rb.Spec.RoleID), nil)
	}
	role, err := s.roleStore.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.Scope != ScopeNamespace {
		return nil, apierrors.NewBadRequest("role scope must be 'namespace' for namespace-level bindings", nil)
	}
	if role.NamespaceID == nil || *role.NamespaceID != nsID {
		return nil, apierrors.NewBadRequest("role does not belong to this namespace", nil)
	}

	userID, err := parseID(rb.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", rb.Spec.UserID), nil)
	}

	if options.DryRun {
		return rb, nil
	}

	created, err := s.rbStore.Create(ctx, &DBRoleBinding{
		UserID:      userID,
		RoleID:      roleID,
		Scope:       ScopeNamespace,
		WorkspaceID: &wsID,
		NamespaceID: &nsID,
	})
	if err != nil {
		return nil, err
	}

	sharedPermCache.Invalidate(userID)

	return roleBindingToAPI(created, "", "", role.Name, role.DisplayName), nil
}

// +openapi:summary=删除项目级角色绑定
// +openapi:summary.workspaces.namespaces.rolebindings=删除工作空间下项目的角色绑定
func (s *namespaceRoleBindingStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["rolebindingId"]
	rbID, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role binding ID: %s", id), nil)
	}

	existing, err := s.rbStore.GetByID(ctx, rbID)
	if err != nil {
		return err
	}
	if existing.IsOwner {
		return apierrors.NewBadRequest("cannot delete owner role binding", nil)
	}

	if options.DryRun {
		return nil
	}

	if err := s.rbStore.Delete(ctx, rbID); err != nil {
		return err
	}

	sharedPermCache.Invalidate(existing.UserID)

	return nil
}

// roleBindingToAPI converts a DBRoleBinding to the API type with optional display info.
func roleBindingToAPI(rb *DBRoleBinding, username, userDisplayName, roleName, roleDisplayName string) *RoleBinding {
	var wsID, nsID *string
	if rb.WorkspaceID != nil {
		s := strconv.FormatInt(*rb.WorkspaceID, 10)
		wsID = &s
	}
	if rb.NamespaceID != nil {
		s := strconv.FormatInt(*rb.NamespaceID, 10)
		nsID = &s
	}
	return &RoleBinding{
		TypeMeta: runtime.TypeMeta{Kind: "RoleBinding"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(rb.ID, 10),
			CreatedAt: &rb.CreatedAt,
		},
		Spec: RoleBindingSpec{
			UserID:          strconv.FormatInt(rb.UserID, 10),
			RoleID:          strconv.FormatInt(rb.RoleID, 10),
			Scope:           rb.Scope,
			WorkspaceID:     wsID,
			NamespaceID:     nsID,
			IsOwner:         rb.IsOwner,
			RoleName:        roleName,
			RoleDisplayName: roleDisplayName,
			Username:        username,
			UserDisplayName: userDisplayName,
		},
	}
}

func roleBindingWithDetailsToAPI(rb *DBRoleBindingWithDetails) *RoleBinding {
	return roleBindingToAPI(&rb.RoleBinding, rb.Username, rb.UserDisplayName, rb.RoleName, rb.RoleDisplayName)
}

func roleBindingListToAPI(result *db.ListResult[DBRoleBindingWithDetails]) *RoleBindingList {
	items := make([]RoleBinding, len(result.Items))
	for i, item := range result.Items {
		items[i] = *roleBindingWithDetailsToAPI(&item)
	}
	return &RoleBindingList{
		TypeMeta:   runtime.TypeMeta{Kind: "RoleBindingList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}
}

func permissionToAPI(p *DBPermission) *Permission {
	return &Permission{
		TypeMeta: runtime.TypeMeta{Kind: "Permission"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(p.ID, 10),
			CreatedAt: &p.CreatedAt,
		},
		Spec: PermissionSpec{
			Code:        p.Code,
			Method:      p.Method,
			Path:        p.Path,
			Scope:       p.Scope,
			Description: p.Description,
		},
	}
}
