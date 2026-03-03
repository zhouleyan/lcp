package store

import (
	"lcp.io/lcp/lib/db/generated"
	libstore "lcp.io/lcp/lib/store"
)

// Namespace is an alias for the sqlc-generated Namespace model.
type Namespace = generated.Namespace

// UserNamespaceRole is an alias for the sqlc-generated UserNamespace model.
type UserNamespaceRole = generated.UserNamespace

// NamespaceWithOwner extends Namespace with owner username.
type NamespaceWithOwner = libstore.NamespaceWithOwner

// NamespaceWithRole is a namespace with the user's role in it.
type NamespaceWithRole = libstore.NamespaceWithRole

// UserWithRole is a user with their role in a namespace.
type UserWithRole = libstore.UserWithRole
