package iam

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/logger"
)

// SeedAdmin ensures the initial admin user exists. If the user already exists, it is a no-op.
func SeedAdmin(ctx context.Context, userStore UserStore, cfg config.AdminConfig) error {
	_, err := userStore.GetByUsername(ctx, cfg.Username)
	if err == nil {
		return nil // admin already exists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), 10)
	if err != nil {
		return err
	}

	user, err := userStore.Create(ctx, &DBUser{
		Username:    cfg.Username,
		Email:       cfg.Email,
		DisplayName: cfg.DisplayName,
		Phone:       cfg.Phone,
		Status:      "active",
	})
	if err != nil {
		return err
	}

	if err := userStore.SetPasswordHash(ctx, user.ID, string(hash)); err != nil {
		return err
	}

	logger.Infof("seeded admin user — username: %s, password: %s", cfg.Username, cfg.Password)
	return nil
}
