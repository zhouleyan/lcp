package iam

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/oidc"
)

// userLookupAdapter adapts iam.UserStore to oidc.UserLookup.
type userLookupAdapter struct {
	store UserStore
}

// NewUserLookupAdapter creates a UserLookup adapter.
func NewUserLookupAdapter(store UserStore) oidc.UserLookup {
	return &userLookupAdapter{store: store}
}

func (a *userLookupAdapter) GetByIdentifier(ctx context.Context, identifier string) (*oidc.OIDCUser, error) {
	row, err := a.store.GetUserForAuth(ctx, identifier)
	if err != nil {
		return nil, err
	}
	return &oidc.OIDCUser{
		ID:           row.ID,
		Username:     row.Username,
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		Phone:        row.Phone,
		PasswordHash: row.PasswordHash,
		Status:       row.Status,
	}, nil
}

func (a *userLookupAdapter) GetByID(ctx context.Context, id int64) (*oidc.OIDCUser, error) {
	user, err := a.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &oidc.OIDCUser{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Phone:       user.Phone,
		Status:      user.Status,
	}, nil
}

func (a *userLookupAdapter) UpdateLastLogin(ctx context.Context, id int64) error {
	return a.store.UpdateLastLogin(ctx, id)
}

// refreshTokenAdapter adapts iam.RefreshTokenStore to oidc.RefreshTokenStore.
type refreshTokenAdapter struct {
	store RefreshTokenStore
}

// NewRefreshTokenAdapter creates a RefreshTokenStore adapter.
func NewRefreshTokenAdapter(store RefreshTokenStore) oidc.RefreshTokenStore {
	return &refreshTokenAdapter{store: store}
}

func (a *refreshTokenAdapter) Store(ctx context.Context, rt *oidc.RefreshTokenData) error {
	_, err := a.store.Create(ctx, &DBRefreshToken{
		TokenHash: rt.TokenHash,
		UserID:    rt.UserID,
		ClientID:  rt.ClientID,
		Scope:     rt.Scope,
		ExpiresAt: rt.ExpiresAt,
	})
	return err
}

func (a *refreshTokenAdapter) Consume(ctx context.Context, rawToken string) (*oidc.RefreshTokenData, error) {
	tokenHash := oidc.HashToken(rawToken)

	// Atomic consume: UPDATE ... WHERE revoked=false AND expires_at > now() RETURNING *
	rt, err := a.store.ConsumeByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	return &oidc.RefreshTokenData{
		TokenHash: rt.TokenHash,
		UserID:    rt.UserID,
		ClientID:  rt.ClientID,
		Scope:     rt.Scope,
		ExpiresAt: rt.ExpiresAt,
	}, nil
}

func (a *refreshTokenAdapter) RevokeByUser(ctx context.Context, userID int64) error {
	return a.store.RevokeByUserID(ctx, userID)
}
