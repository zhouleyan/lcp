package user

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"
)

// userStorage implements rest.StandardStorage with validation internalized.
type userStorage struct {
	svc *UserService
}

func newUserStorage(svc *UserService) rest.StandardStorage {
	return &userStorage{svc: svc}
}

// Get implements rest.Getter.
func (s *userStorage) Get(ctx context.Context, id string) (runtime.Object, error) {
	return s.svc.GetUser(ctx, id)
}

// List implements rest.Lister.
func (s *userStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	pagination := store.Pagination{
		Page:     options.Pagination.Page,
		PageSize: options.Pagination.PageSize,
	}
	return s.svc.ListUsers(ctx, options.Filters, pagination, options.SortBy, string(options.SortOrder))
}

// Create implements rest.Creator with internal validation.
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

	return s.svc.CreateUser(ctx, user)
}

// Update implements rest.Updater with internal validation.
func (s *userStorage) Update(ctx context.Context, id string, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
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

	return s.svc.UpdateUser(ctx, id, user)
}

// Patch implements rest.Patcher.
func (s *userStorage) Patch(ctx context.Context, id string, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	user, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", obj)
	}

	if options.DryRun {
		existing, err := s.svc.GetUser(ctx, id)
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	return s.svc.PatchUser(ctx, id, user)
}

// Delete implements rest.Deleter.
func (s *userStorage) Delete(ctx context.Context, id string, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}
	return s.svc.DeleteUser(ctx, id)
}

// DeleteCollection implements rest.CollectionDeleter.
func (s *userStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	result, err := s.svc.DeleteUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: result.SuccessCount,
		FailedCount:  result.FailedCount,
		FailedIDs:    result.FailedIDs,
	}, nil
}
