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
		return nil, fmt.Errorf("get user: %w", err)
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
		return nil, fmt.Errorf("list users: %w", err)
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
		return nil, fmt.Errorf("create user: %w", err)
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
		return nil, fmt.Errorf("update user: %w", err)
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
		return nil, fmt.Errorf("patch user: %w", err)
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
		return fmt.Errorf("delete user: %w", err)
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
		return nil, fmt.Errorf("delete users: %w", err)
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// ===== namespaceStorage =====

// namespaceStorage implements rest.Getter, rest.Lister, rest.Creator,
// rest.Updater, rest.Patcher, rest.Deleter for namespaces.
type namespaceStorage struct {
	nsStore   NamespaceStore
	userStore UserStore
}

func (s *namespaceStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	//TODO implement me
	panic("implement me")
}

// NewNamespaceStorage creates a new REST storage for namespaces.
func NewNamespaceStorage(nsStore NamespaceStore, userStore UserStore) rest.StandardStorage {
	return &namespaceStorage{nsStore: nsStore, userStore: userStore}
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
		return nil, fmt.Errorf("get namespace: %w", err)
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

	result, err := s.nsStore.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
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

	if errs := ValidateNamespaceCreate(ns.ObjectMeta.Name, &ns.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ns, nil
	}

	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	// Check owner exists
	if _, err := s.userStore.GetByID(ctx, ownerID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("owner user %d not found", ownerID), nil)
	}

	created, err := s.nsStore.Create(ctx, &DBNamespace{
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
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

	updated, err := s.nsStore.Update(ctx, &DBNamespace{
		ID:          nid,
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
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
		return nil, fmt.Errorf("get namespace for patch: %w", err)
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
		return nil, fmt.Errorf("patch namespace: %w", err)
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
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

// ===== memberStorage =====

// memberStorage implements rest.Creator and rest.Lister for the members sub-resource.
type memberStorage struct {
	nsStore   NamespaceStore
	unStore   UserNamespaceStore
	userStore UserStore
}

func (s *memberStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *memberStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *memberStorage) Patch(ctx context.Context, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (s *memberStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	//TODO implement me
	panic("implement me")
}

func (s *memberStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	//TODO implement me
	panic("implement me")
}

// NewMemberStorage creates a new REST storage for namespace members.
func NewMemberStorage(nsStore NamespaceStore, unStore UserNamespaceStore, userStore UserStore) rest.StandardStorage {
	return &memberStorage{nsStore: nsStore, unStore: unStore, userStore: userStore}
}

func (s *memberStorage) NewObject() runtime.Object { return &NamespaceMember{} }

func (s *memberStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	namespaceID := options.PathParams["namespaceId"]
	if namespaceID == "" {
		return nil, apierrors.NewBadRequest("missing namespaceId in path", nil)
	}

	member, ok := obj.(*NamespaceMember)
	if !ok {
		return nil, fmt.Errorf("expected *NamespaceMember, got %T", obj)
	}

	if errs := ValidateNamespaceMember(&member.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return member, nil
	}

	nsID, err := parseID(namespaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", namespaceID), nil)
	}

	userID, err := parseID(member.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid userId: %s", member.Spec.UserID), nil)
	}

	// Check user exists
	if _, err := s.userStore.GetByID(ctx, userID); err != nil {
		return nil, apierrors.NewNotFound("User", member.Spec.UserID)
	}

	// Check namespace exists
	if _, err := s.nsStore.GetByID(ctx, nsID); err != nil {
		return nil, apierrors.NewNotFound("Namespace", namespaceID)
	}

	role, err := s.unStore.Add(ctx, &DBUserNamespace{
		UserID:      userID,
		NamespaceID: nsID,
		Role:        member.Spec.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return memberToAPI(role), nil
}

func (s *memberStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	namespaceID := options.PathParams["namespaceId"]
	if namespaceID == "" {
		return nil, apierrors.NewBadRequest("missing namespaceId in path", nil)
	}

	nsID, err := parseID(namespaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", namespaceID), nil)
	}

	members, err := s.unStore.ListByNamespaceID(ctx, nsID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	items := make([]NamespaceMember, len(members))
	for i, m := range members {
		items[i] = NamespaceMember{
			TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "NamespaceMember"},
			Spec: NamespaceMemberSpec{
				UserID: fmt.Sprintf("%d", m.User.ID),
				Role:   m.Role,
			},
		}
	}

	return &NamespaceMemberList{
		TypeMeta: runtime.TypeMeta{Kind: "NamespaceMemberList", APIVersion: "v1"},
		Items:    items,
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
	return userToAPI(&u.User)
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
			OwnerID:     strconv.FormatInt(n.OwnerID, 10),
			Visibility:  n.Visibility,
			MaxMembers:  int(n.MaxMembers),
			Status:      n.Status,
		},
	}
}

func memberToAPI(r *DBUserNamespace) *NamespaceMember {
	return &NamespaceMember{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "NamespaceMember"},
		Spec: NamespaceMemberSpec{
			UserID: strconv.FormatInt(r.UserID, 10),
			Role:   r.Role,
		},
	}
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
