package user

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"

	userstore "lcp.io/lcp/pkg/modules/user/store"
)

// UserService handles user business logic.
type UserService struct {
	store userstore.UserStore
}

// NewUserService creates a new UserService.
func NewUserService(s userstore.UserStore) *UserService {
	return &UserService{store: s}
}

func (u *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
	created, err := u.store.Create(ctx, &userstore.User{
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

	user, err := u.store.GetByID(ctx, uid)
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
	for k, v := range filters {
		query.Filters[k] = v
	}
	if sortBy != "" {
		query.SortBy = sortBy
	}
	if sortOrder != "" {
		query.SortOrder = sortOrder
	}

	result, err := u.store.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]User, len(result.Items))
	for i, item := range result.Items {
		items[i] = *userWithNamespacesToAPI(&item)
	}

	return &UserList{
		TypeMeta: runtime.TypeMeta{Kind: "UserList", APIVersion: "v1"},
		Items:    items,
		TotalCount: result.TotalCount,
	}, nil
}

func (u *UserService) UpdateUser(ctx context.Context, id string, user *User) (*User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	updated, err := u.store.Update(ctx, &userstore.User{
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

func (u *UserService) PatchUser(ctx context.Context, id string, user *User) (*User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	patched, err := u.store.Patch(ctx, uid, &userstore.User{
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

	if err := u.store.Delete(ctx, uid); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// DeletionResult holds batch delete results.
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
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

	count, err := u.store.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, fmt.Errorf("delete users: %w", err)
	}

	return &DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}
