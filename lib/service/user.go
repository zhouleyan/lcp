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

	created, err := u.s.store.Users().Create(ctx, store.CreateUserParams{
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
