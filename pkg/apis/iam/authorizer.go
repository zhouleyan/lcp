package iam

import (
	"time"

	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
)

// NewAuthorizer creates a fully-wired Authorizer from IAM stores and API group definitions.
func NewAuthorizer(rbStore RoleBindingStore, nsStore NamespaceStore, groups []*rest.APIGroupInfo) *filters.Authorizer {
	lookup := BuildPermissionLookup(groups)
	checker := NewRBACChecker(rbStore)
	nsResolver := NewNamespaceResolver(nsStore, 10*time.Minute)

	return &filters.Authorizer{
		Lookup:     lookup,
		Checker:    checker,
		NSResolver: nsResolver,
	}
}
