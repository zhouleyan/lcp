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
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// PasswordHasher 密码哈希函数类型。
type PasswordHasher func(password string) (string, error)

// ===== userStorage 用户存储 =====

// userStorage 用户资源的 REST 存储实现，支持 CRUD、批量删除和密码管理。
type userStorage struct {
	dbStore    UserStore
	uwStore    UserWorkspaceStore
	unStore    UserNamespaceStore
	hashPasswd PasswordHasher
}

// NewUserStorage 创建用户 REST 存储（无密码功能）。
func NewUserStorage(dbStore UserStore, uwStore UserWorkspaceStore, unStore UserNamespaceStore) rest.StandardStorage {
	return &userStorage{dbStore: dbStore, uwStore: uwStore, unStore: unStore}
}

// NewUserStorageWithPassword 创建支持密码哈希的用户 REST 存储。
func NewUserStorageWithPassword(dbStore UserStore, uwStore UserWorkspaceStore, unStore UserNamespaceStore, hashPasswd PasswordHasher) rest.StandardStorage {
	return &userStorage{dbStore: dbStore, uwStore: uwStore, unStore: unStore, hashPasswd: hashPasswd}
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

	result := userToAPI(user)

	// Enrich with associated workspaces
	if s.uwStore != nil {
		if wsItems, err := s.uwStore.ListByUserID(ctx, uid); err != nil {
			logger.Infof("failed to enrich user %d with workspaces: %v", uid, err)
		} else {
			for _, ws := range wsItems {
				result.Spec.Workspaces = append(result.Spec.Workspaces, UserWorkspaceRef{
					ID:          strconv.FormatInt(ws.ID, 10),
					Name:        ws.Name,
					DisplayName: ws.DisplayName,
					Role:        ws.Role,
					JoinedAt:    ws.JoinedAt.Format(time.RFC3339),
				})
			}
		}
	}

	// Enrich with associated namespaces
	if s.unStore != nil {
		if nsItems, err := s.unStore.ListByUserID(ctx, uid); err != nil {
			logger.Infof("failed to enrich user %d with namespaces: %v", uid, err)
		} else {
			for _, ns := range nsItems {
				result.Spec.NamespaceRefs = append(result.Spec.NamespaceRefs, UserNamespaceRef{
					ID:          strconv.FormatInt(ns.ID, 10),
					Name:        ns.Name,
					DisplayName: ns.DisplayName,
					WorkspaceID: strconv.FormatInt(ns.WorkspaceID, 10),
					Role:        ns.Role,
					JoinedAt:    ns.JoinedAt.Format(time.RFC3339),
				})
			}
		}
	}

	return result, nil
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
}

// NewWorkspaceStorage 创建工作空间 REST 存储。
func NewWorkspaceStorage(wsStore WorkspaceStore, userStore UserStore) rest.StandardStorage {
	return &workspaceStorage{wsStore: wsStore, userStore: userStore}
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

	// Re-fetch to get full response with ownerName, namespaceCount, memberCount
	full, err := s.wsStore.GetByID(ctx, created.ID)
	if err != nil {
		return workspaceToAPI(created), nil
	}
	return workspaceWithOwnerToAPI(full), nil
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

	existingWithOwner, err := s.wsStore.GetByID(ctx, wid)
	if err != nil {
		return nil, err
	}

	existing := &existingWithOwner.Workspace
	if ws.ObjectMeta.Name != "" {
		existing.Name = ws.ObjectMeta.Name
	}
	if ws.Spec.DisplayName != "" {
		existing.DisplayName = ws.Spec.DisplayName
	}
	if ws.Spec.Description != "" {
		existing.Description = ws.Spec.Description
	}
	if ws.Spec.OwnerID != "" {
		ownerID, err := parseID(ws.Spec.OwnerID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ws.Spec.OwnerID), nil)
		}
		existing.OwnerID = ownerID
	}
	if ws.Spec.Status != "" {
		existing.Status = ws.Spec.Status
	}

	updated, err := s.wsStore.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	return workspaceToAPI(updated), nil
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

	if err := s.wsStore.Delete(ctx, wid); err != nil {
		return err
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
}

// NewNamespaceStorage 创建项目 REST 存储。
func NewNamespaceStorage(nsStore NamespaceStore, wsStore WorkspaceStore, userStore UserStore) rest.StandardStorage {
	return &namespaceStorage{nsStore: nsStore, wsStore: wsStore, userStore: userStore}
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

	// Re-fetch to get enriched response with ownerName, memberCount
	full, err := s.nsStore.GetByID(ctx, created.ID)
	if err != nil {
		return namespaceToAPI(created), nil
	}
	return namespaceWithOwnerToAPI(full), nil
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

	// Fetch existing and merge
	existingWithOwner, err := s.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}
	existing := &existingWithOwner.Namespace

	if ns.ObjectMeta.Name != "" {
		existing.Name = ns.ObjectMeta.Name
	}
	if ns.Spec.DisplayName != "" {
		existing.DisplayName = ns.Spec.DisplayName
	}
	if ns.Spec.Description != "" {
		existing.Description = ns.Spec.Description
	}
	if ns.Spec.WorkspaceID != "" {
		workspaceID, err := parseID(ns.Spec.WorkspaceID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspaceId: %s", ns.Spec.WorkspaceID), nil)
		}
		existing.WorkspaceID = workspaceID
	}
	if ns.Spec.OwnerID != "" {
		ownerID, err := parseID(ns.Spec.OwnerID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
		}
		existing.OwnerID = ownerID
	}
	if ns.Spec.Visibility != "" {
		existing.Visibility = ns.Spec.Visibility
	}
	if ns.Spec.MaxMembers != 0 {
		existing.MaxMembers = int32(ns.Spec.MaxMembers)
	}
	if ns.Spec.Status != "" {
		existing.Status = ns.Spec.Status
	}

	updated, err := s.nsStore.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	return namespaceToAPI(updated), nil
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

	if err := s.nsStore.Delete(ctx, nid); err != nil {
		return err
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
	uwStore   UserWorkspaceStore
	userStore UserStore
}

// NewWorkspaceUserStorage 创建工作空间成员管理 REST 存储。
func NewWorkspaceUserStorage(uwStore UserWorkspaceStore, userStore UserStore) rest.Storage {
	return &workspaceUserStorage{uwStore: uwStore, userStore: userStore}
}

func (s *workspaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

// +openapi:summary=获取工作空间成员列表
func (s *workspaceUserStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

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

	result, err := s.uwStore.ListByWorkspaceID(ctx, wsID, query)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(result.Items))
	for i, m := range result.Items {
		items[i] = *userToAPI(&m.User)
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

	added := 0
	for _, idStr := range req.IDs {
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		// Verify user exists
		if _, err := s.userStore.GetByID(ctx, uid); err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("user %s not found", idStr), nil)
		}
		result, err := s.uwStore.Add(ctx, &DBUserWorkspace{
			UserID:      uid,
			WorkspaceID: wsID,
			Role:        "member",
		})
		if err != nil {
			return nil, err
		}
		if result != nil {
			added++
		}
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

	for _, idStr := range ids {
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		if err := s.uwStore.Remove(ctx, uid, wsID); err != nil {
			return nil, err
		}
	}

	return &rest.DeletionResult{
		SuccessCount: len(ids),
	}, nil
}

// ===== namespaceUserStorage 项目成员存储 =====

// namespaceUserStorage 管理项目的成员关系，支持查询成员列表、批量添加和批量移除成员。
// 添加项目成员时会自动将其加入父工作空间。
// +openapi:path=/workspaces/{workspaceId}/namespaces/{namespaceId}/users
type namespaceUserStorage struct {
	unStore   UserNamespaceStore
	nsStore   NamespaceStore
	userStore UserStore
}

// NewNamespaceUserStorage 创建项目成员管理 REST 存储。
func NewNamespaceUserStorage(unStore UserNamespaceStore, nsStore NamespaceStore, userStore UserStore) rest.Storage {
	return &namespaceUserStorage{unStore: unStore, nsStore: nsStore, userStore: userStore}
}

func (s *namespaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

// +openapi:summary=获取项目成员列表
// +openapi:summary.workspaces.namespaces.users=获取工作空间下项目的成员列表
func (s *namespaceUserStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	members, err := s.unStore.ListByNamespaceID(ctx, nsID)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(members))
	for i, m := range members {
		items[i] = *userToAPI(&m.User)
	}

	return &UserList{
		TypeMeta:   runtime.TypeMeta{Kind: "UserList"},
		Items:      items,
		TotalCount: int64(len(items)),
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

	added := 0
	for _, idStr := range req.IDs {
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		// Verify user exists
		if _, err := s.userStore.GetByID(ctx, uid); err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("user %s not found", idStr), nil)
		}
		// Add will auto-add to workspace via transaction in store
		result, err := s.unStore.Add(ctx, &DBUserNamespace{
			UserID:      uid,
			NamespaceID: nsID,
			Role:        "member",
		})
		if err != nil {
			return nil, err
		}
		if result != nil {
			added++
		}
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

	for _, idStr := range ids {
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		if err := s.unStore.Remove(ctx, uid, nsID); err != nil {
			return nil, err
		}
	}

	return &rest.DeletionResult{
		SuccessCount: len(ids),
	}, nil
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

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
