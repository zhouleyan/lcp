package service

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/api/validation"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"
)

type UserService struct {
	s *Service
}

func (u *UserService) CreateUser(ctx context.Context, user *types.User) (*types.User, error) {
	if errs := validation.ValidateUserCreate(&user.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	created, err := u.s.store.Users().Create(ctx, &store.User{
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

func (u *UserService) GetUser(ctx context.Context, id string) (runtime.Object, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	user, err := u.s.store.Users().GetByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return userToAPI(user), nil
}

func (u *UserService) ListUsers(ctx context.Context, filters map[string]string, pagination store.Pagination, sortBy, sortOrder string) (runtime.Object, error) {
	query := store.ListQuery{
		Filters:    make(map[string]any),
		Pagination: pagination,
	}

	// Convert string filters to any
	for k, v := range filters {
		query.Filters[k] = v
	}

	// Override pagination fields if provided
	if sortBy != "" {
		query.SortBy = sortBy
	}
	if sortOrder != "" {
		query.SortOrder = sortOrder
	}

	result, err := u.s.store.Users().List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]types.User, len(result.Items))
	for i, item := range result.Items {
		items[i] = *userWithNamespacesToAPI(&item)
	}

	return &types.UserList{
		TypeMeta: runtime.TypeMeta{
			Kind:       "UserList",
			APIVersion: "v1",
		},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

func (u *UserService) UpdateUser(ctx context.Context, id string, user *types.User) (*types.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	updated, err := u.s.store.Users().Update(ctx, &store.User{
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

func (u *UserService) PatchUser(ctx context.Context, id string, user *types.User) (*types.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	patched, err := u.s.store.Users().Patch(ctx, uid, &store.User{
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

func (u *UserService) DeleteUser(ctx context.Context, id string) error {
	uid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	if err := u.s.store.Users().Delete(ctx, uid); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func (u *UserService) DeleteUsers(ctx context.Context, ids []string) (*DeletionResult, error) {
	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		uid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, uid)
	}

	count, err := u.s.store.Users().DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, fmt.Errorf("delete users: %w", err)
	}

	return &DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// DeletionResult 批量删除结果
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
}

func userWithNamespacesToAPI(u *store.UserWithNamespaces) *types.User {
	user := userToAPI(&u.User)
	// 可以在这里添加 namespace 信息到 user 对象
	return user
}
