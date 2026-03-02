package handler

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/service"
)

type userStorage struct {
	svc *service.Service
}

func newUserStorage(svc *service.Service) rest.StandardStorage {
	return &userStorage{svc: svc}
}

// Get 实现 rest.Getter
func (s *userStorage) Get(ctx context.Context, id string) (runtime.Object, error) {
	return s.svc.Users().GetUser(ctx, id)
}

// List 实现 rest.Lister
func (s *userStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	return s.svc.Users().ListUsers(ctx, options.Filters, options.Pagination, options.SortBy, options.SortOrder)
}

// Create 实现 rest.Creater
func (s *userStorage) Create(ctx context.Context, obj runtime.Object, validate rest.ValidateObjectFunc, options *rest.CreateOptions) (runtime.Object, error) {
	user, ok := obj.(*types.User)
	if !ok {
		return nil, fmt.Errorf("expected *types.User, got %T", obj)
	}

	if validate != nil {
		if err := validate(ctx, obj); err != nil {
			return nil, err
		}
	}

	if options.DryRun {
		return user, nil
	}

	return s.svc.Users().CreateUser(ctx, user)
}

// Update 实现 rest.Updater
func (s *userStorage) Update(ctx context.Context, id string, obj runtime.Object, validate rest.ValidateObjectFunc, options *rest.UpdateOptions) (runtime.Object, error) {
	user, ok := obj.(*types.User)
	if !ok {
		return nil, fmt.Errorf("expected *types.User, got %T", obj)
	}

	if validate != nil {
		if err := validate(ctx, obj); err != nil {
			return nil, err
		}
	}

	if options.DryRun {
		return user, nil
	}

	return s.svc.Users().UpdateUser(ctx, id, user)
}

// Patch 实现 rest.Patcher
func (s *userStorage) Patch(ctx context.Context, id string, obj runtime.Object, validate rest.ValidateObjectFunc, options *rest.PatchOptions) (runtime.Object, error) {
	user, ok := obj.(*types.User)
	if !ok {
		return nil, fmt.Errorf("expected *types.User, got %T", obj)
	}

	if validate != nil {
		if err := validate(ctx, obj); err != nil {
			return nil, err
		}
	}

	if options.DryRun {
		// 获取现有用户用于预览
		existing, err := s.svc.Users().GetUser(ctx, id)
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	return s.svc.Users().PatchUser(ctx, id, user)
}

// Delete 实现 rest.Deleter
func (s *userStorage) Delete(ctx context.Context, id string, validate rest.ValidateObjectFunc, options *rest.DeleteOptions) error {
	if validate != nil {
		// 获取用户用于验证
		user, err := s.svc.Users().GetUser(ctx, id)
		if err != nil {
			return err
		}
		if err := validate(ctx, user); err != nil {
			return err
		}
	}

	if options.DryRun {
		return nil
	}

	return s.svc.Users().DeleteUser(ctx, id)
}

// DeleteCollection 实现 rest.CollectionDeleter
func (s *userStorage) DeleteCollection(ctx context.Context, ids []string, validate rest.ValidateObjectFunc, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if validate != nil {
		// 可以在这里批量验证
		// 简化实现，暂时跳过
	}

	if options.DryRun {
		return &rest.DeletionResult{
			SuccessCount: len(ids),
			FailedCount:  0,
		}, nil
	}

	result, err := s.svc.Users().DeleteUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: result.SuccessCount,
		FailedCount:  result.FailedCount,
		FailedIDs:    result.FailedIDs,
	}, nil
}
