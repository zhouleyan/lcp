package user

import (
	"context"
	"fmt"

	userstore "lcp.io/lcp/pkg/modules/user/store"
)

// Lookup provides a UserLookup implementation for cross-module use.
type Lookup struct {
	store userstore.UserStore
}

// NewLookup creates a new Lookup backed by the given UserStore.
func NewLookup(s userstore.UserStore) *Lookup {
	return &Lookup{store: s}
}

// UserExists checks if a user with the given ID exists.
func (l *Lookup) UserExists(ctx context.Context, userID int64) error {
	_, err := l.store.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user %d not found: %w", userID, err)
	}
	return nil
}
