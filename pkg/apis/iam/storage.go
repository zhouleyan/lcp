package iam

import (
	"context"
	"fmt"
	"strconv"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// ===== userStorage =====

// userStorage implements rest.StandardStorage for users.
type userStorage struct {
	dbStore UserStore
}

// NewUserStorage creates a new REST storage backed by the given UserStore.
func NewUserStorage(dbStore UserStore) rest.StandardStorage {
	return &userStorage{dbStore: dbStore}
}

func (s *userStorage) NewObject() runtime.Object { return &User{} }

// Get implements rest.Getter.
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

// List implements rest.Lister.
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
		TypeMeta:   runtime.TypeMeta{Kind: "UserList", APIVersion: "v1"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// Create implements rest.Creator.
func (s *userStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	user, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", obj)
	}

	if errs := ValidateUserCreate(&user.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
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
	return userToAPI(created), nil
}

// Update implements rest.Updater.
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

// Patch implements rest.Patcher.
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

// Delete implements rest.Deleter.
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

// DeleteCollection implements rest.CollectionDeleter.
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

// ===== workspaceStorage =====

type workspaceStorage struct {
	wsStore   WorkspaceStore
	userStore UserStore
}

// NewWorkspaceStorage creates a new REST storage for workspaces.
func NewWorkspaceStorage(wsStore WorkspaceStore, userStore UserStore) rest.StandardStorage {
	return &workspaceStorage{wsStore: wsStore, userStore: userStore}
}

func (s *workspaceStorage) NewObject() runtime.Object { return &Workspace{} }

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
	return workspaceToAPI(ws), nil
}

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
		items[i] = *workspaceToAPI(&item.Workspace)
	}

	return &WorkspaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "WorkspaceList", APIVersion: "v1"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

func (s *workspaceStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ws, ok := obj.(*Workspace)
	if !ok {
		return nil, fmt.Errorf("expected *Workspace, got %T", obj)
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

	return workspaceToAPI(created), nil
}

func (s *workspaceStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ws, ok := obj.(*Workspace)
	if !ok {
		return nil, fmt.Errorf("expected *Workspace, got %T", obj)
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

	existing, err := s.wsStore.GetByID(ctx, wid)
	if err != nil {
		return nil, err
	}

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

// ===== namespaceStorage =====

// namespaceStorage implements rest.StandardStorage for namespaces.
type namespaceStorage struct {
	nsStore   NamespaceStore
	wsStore   WorkspaceStore
	userStore UserStore
}

// NewNamespaceStorage creates a new REST storage for namespaces.
func NewNamespaceStorage(nsStore NamespaceStore, wsStore WorkspaceStore, userStore UserStore) rest.StandardStorage {
	return &namespaceStorage{nsStore: nsStore, wsStore: wsStore, userStore: userStore}
}

func (s *namespaceStorage) NewObject() runtime.Object { return &Namespace{} }

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

	return namespaceToAPI(ns), nil
}

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
		items[i] = *namespaceToAPI(&item.Namespace)
	}

	return &NamespaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "NamespaceList", APIVersion: "v1"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

func (s *namespaceStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	// If workspace ID comes from path params, use it
	if wsID, ok := options.PathParams["workspaceId"]; ok && wsID != "" {
		ns.Spec.WorkspaceID = wsID
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

	return namespaceToAPI(created), nil
}

func (s *namespaceStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
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
	existing, err := s.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

	workspaceID := existing.WorkspaceID
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
	existing, err := s.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, err
	}

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

// ===== workspaceUserStorage =====

// workspaceUserStorage manages users within a workspace (batch operations).
type workspaceUserStorage struct {
	uwStore   UserWorkspaceStore
	userStore UserStore
}

// NewWorkspaceUserStorage creates a new REST storage for workspace user management.
func NewWorkspaceUserStorage(uwStore UserWorkspaceStore, userStore UserStore) rest.Storage {
	return &workspaceUserStorage{uwStore: uwStore, userStore: userStore}
}

func (s *workspaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

func (s *workspaceUserStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	members, err := s.uwStore.ListByWorkspaceID(ctx, wsID)
	if err != nil {
		return nil, err
	}

	items := make([]User, len(members))
	for i, m := range members {
		items[i] = *userToAPI(&m.User)
	}

	return &UserList{
		TypeMeta:   runtime.TypeMeta{Kind: "UserList", APIVersion: "v1"},
		Items:      items,
		TotalCount: int64(len(items)),
	}, nil
}

func (s *workspaceUserStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
	}

	req, ok := obj.(*BatchRequest)
	if !ok {
		return nil, fmt.Errorf("expected *BatchRequest, got %T", obj)
	}

	for _, idStr := range req.IDs {
		uid, err := parseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", idStr), nil)
		}
		// Verify user exists
		if _, err := s.userStore.GetByID(ctx, uid); err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("user %s not found", idStr), nil)
		}
		_, err = s.uwStore.Add(ctx, &DBUserWorkspace{
			UserID:      uid,
			WorkspaceID: wsID,
			Role:        "member",
		})
		if err != nil {
			return nil, err
		}
	}

	return &rest.DeletionResult{
		TypeMeta:     runtime.TypeMeta{Kind: "Result", APIVersion: "v1"},
		SuccessCount: len(req.IDs),
	}, nil
}

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

// ===== namespaceUserStorage =====

// namespaceUserStorage manages users within a namespace (batch operations).
type namespaceUserStorage struct {
	unStore   UserNamespaceStore
	userStore UserStore
}

// NewNamespaceUserStorage creates a new REST storage for namespace user management.
func NewNamespaceUserStorage(unStore UserNamespaceStore, userStore UserStore) rest.Storage {
	return &namespaceUserStorage{unStore: unStore, userStore: userStore}
}

func (s *namespaceUserStorage) NewObject() runtime.Object { return &BatchRequest{} }

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
		TypeMeta:   runtime.TypeMeta{Kind: "UserList", APIVersion: "v1"},
		Items:      items,
		TotalCount: int64(len(items)),
	}, nil
}

func (s *namespaceUserStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
	}

	req, ok := obj.(*BatchRequest)
	if !ok {
		return nil, fmt.Errorf("expected *BatchRequest, got %T", obj)
	}

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
		_, err = s.unStore.Add(ctx, &DBUserNamespace{
			UserID:      uid,
			NamespaceID: nsID,
			Role:        "member",
		})
		if err != nil {
			return nil, err
		}
	}

	return &rest.DeletionResult{
		TypeMeta:     runtime.TypeMeta{Kind: "Result", APIVersion: "v1"},
		SuccessCount: len(req.IDs),
	}, nil
}

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

// ===== helpers =====

func userToAPI(u *DBUser) *User {
	return &User{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "User"},
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
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Workspace"},
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

func namespaceToAPI(n *DBNamespace) *Namespace {
	return &Namespace{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
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
