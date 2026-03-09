package iam

import (
	"time"

	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
)

// NewAuthorizer creates a fully-wired Authorizer from IAM stores and API group definitions.
// Internal implementation details (caches, TTLs, store wiring) are encapsulated here.
func NewAuthorizer(rbStore RoleBindingStore, nsStore NamespaceStore, groups []*rest.APIGroupInfo) *filters.Authorizer {
	lookup := BuildPermissionLookup(groups)
	permCache := NewPermissionCache(5 * time.Minute)
	checker := NewRBACChecker(rbStore, permCache)
	nsResolver := NewNamespaceResolver(nsStore, 10*time.Minute)

	return &filters.Authorizer{
		Lookup:     lookup,
		Checker:    checker,
		NSResolver: nsResolver,
	}
}
