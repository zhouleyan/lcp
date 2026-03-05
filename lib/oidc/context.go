package oidc

import (
	"context"
	"net/http"
)

type contextKey int

const userIDKey contextKey = iota

// WithUserID returns a new request with the user ID stored in context.
func WithUserID(r *http.Request, id int64) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userIDKey, id))
}

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
