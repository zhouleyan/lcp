package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db/generated"
)

type pgRefreshTokenStore struct {
	queries *generated.Queries
}

// NewPGRefreshTokenStore creates a new PostgreSQL-backed RefreshTokenStore.
func NewPGRefreshTokenStore(queries *generated.Queries) iam.RefreshTokenStore {
	return &pgRefreshTokenStore{queries: queries}
}

func (s *pgRefreshTokenStore) Create(ctx context.Context, token *iam.DBRefreshToken) (*iam.DBRefreshToken, error) {
	row, err := s.queries.CreateRefreshToken(ctx, generated.CreateRefreshTokenParams{
		TokenHash: token.TokenHash,
		UserID:    token.UserID,
		ClientID:  token.ClientID,
		Scope:     token.Scope,
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	return new(row), nil
}

func (s *pgRefreshTokenStore) GetByHash(ctx context.Context, tokenHash string) (*iam.DBRefreshToken, error) {
	row, err := s.queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("refresh_token", tokenHash)
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return new(row), nil
}

func (s *pgRefreshTokenStore) ConsumeByHash(ctx context.Context, tokenHash string) (*iam.DBRefreshToken, error) {
	row, err := s.queries.ConsumeRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("refresh_token", tokenHash)
		}
		return nil, fmt.Errorf("consume refresh token: %w", err)
	}
	return new(row), nil
}

func (s *pgRefreshTokenStore) Revoke(ctx context.Context, tokenHash string) error {
	if err := s.queries.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (s *pgRefreshTokenStore) RevokeByUserID(ctx context.Context, userID int64) error {
	if err := s.queries.RevokeRefreshTokensByUserID(ctx, userID); err != nil {
		return fmt.Errorf("revoke refresh tokens by user: %w", err)
	}
	return nil
}

func (s *pgRefreshTokenStore) DeleteExpired(ctx context.Context) error {
	if err := s.queries.DeleteExpiredRefreshTokens(ctx); err != nil {
		return fmt.Errorf("delete expired refresh tokens: %w", err)
	}
	return nil
}
