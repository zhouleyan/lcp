package handler

import (
	"context"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/api/validation"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/runtime"
)

// validateUserCreate 验证用户创建
func validateUserCreate(ctx context.Context, obj runtime.Object) error {
	user, ok := obj.(*types.User)
	if !ok {
		return apierrors.NewBadRequest("invalid object type", nil)
	}

	if errs := validation.ValidateUserCreate(&user.Spec); errs.HasErrors() {
		return apierrors.NewBadRequest("validation failed", errs)
	}

	return nil
}

// validateUserUpdate 验证用户更新
func validateUserUpdate(ctx context.Context, obj runtime.Object) error {
	user, ok := obj.(*types.User)
	if !ok {
		return apierrors.NewBadRequest("invalid object type", nil)
	}

	// 可以添加更新特定的验证逻辑
	if errs := validation.ValidateUserCreate(&user.Spec); errs.HasErrors() {
		return apierrors.NewBadRequest("validation failed", errs)
	}

	return nil
}

// validateUserPatch 验证用户补丁
func validateUserPatch(ctx context.Context, obj runtime.Object) error {
	// Patch 验证可以更宽松，因为是部分更新
	return nil
}

// validateUserDelete 验证用户删除
func validateUserDelete(ctx context.Context, obj runtime.Object) error {
	// 可以添加删除前的检查，比如是否有关联数据
	return nil
}
