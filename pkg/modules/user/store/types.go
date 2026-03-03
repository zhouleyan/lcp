package store

import (
	"lcp.io/lcp/lib/db/generated"
	libstore "lcp.io/lcp/lib/store"
)

// User is an alias for the sqlc-generated User model.
type User = generated.User

// UserWithNamespaces extends User with associated namespace names.
type UserWithNamespaces = libstore.UserWithNamespaces
