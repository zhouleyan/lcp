package filters

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"lcp.io/lcp/lib/oidc"
)

// PermissionLookup resolves a (module, resourceChain, verb) triple to a permission code.
// module → resourceChain → verb → permCode
type PermissionLookup interface {
	Get(module, resourceChain, verb string) string
}

// PermissionChecker checks whether a user has a given permission.
type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID int64, permCode string, scope string, workspaceID, namespaceID int64) (bool, error)
	IsPlatformAdmin(ctx context.Context, userID int64) (bool, error)
	GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error)
	GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error)
}

// AccessFilter holds accessible resource IDs for non-admin users.
// nil means no filter (admin sees everything); empty slice means no access.
type AccessFilter struct {
	WorkspaceIDs []int64
	NamespaceIDs []int64
}

type accessFilterCtxKey struct{}

// WithAccessFilter stores the access filter in the context.
func WithAccessFilter(ctx context.Context, f *AccessFilter) context.Context {
	return context.WithValue(ctx, accessFilterCtxKey{}, f)
}

// AccessFilterFromContext retrieves the access filter from the context.
// Returns nil if no filter is set (admin or auth disabled).
func AccessFilterFromContext(ctx context.Context) *AccessFilter {
	if v, ok := ctx.Value(accessFilterCtxKey{}).(*AccessFilter); ok {
		return v
	}
	return nil
}

// Authorizer bundles the authorization components needed by the middleware chain.
type Authorizer struct {
	Lookup     PermissionLookup
	Checker    PermissionChecker
	NSResolver NamespaceResolver
}

// WithAuthorization returns middleware that checks RBAC permissions on API requests.
// Requests not matching /api/ are passed through. Permissions not in the lookup are allowed.
func WithAuthorization(lookup PermissionLookup, checker PermissionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Non-API requests pass through
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Resolve module, resourceChain, verb from URL
			module, resourceChain, verb := ResolveResourceAndVerb(r.Method, path)
			if module == "" || resourceChain == "" || verb == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Look up the permission code
			permCode := lookup.Get(module, resourceChain, verb)
			if permCode == "" {
				// Permission not registered → allow (e.g. discovery endpoints)
				next.ServeHTTP(w, r)
				return
			}

			// Get authenticated user ID
			userID, ok := oidc.UserIDFromContext(r.Context())
			if !ok {
				forbiddenError(w, "no authenticated user")
				return
			}

			// Self-user access: users can always access their own user resource
			if isSelfUserQuery(path, permCode, userID) {
				next.ServeHTTP(w, r)
				return
			}

			// Platform admin short-circuit
			isAdmin, err := checker.IsPlatformAdmin(r.Context(), userID)
			if err != nil {
				forbiddenError(w, "failed to check permissions")
				return
			}
			if isAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Get scope info from RequestInfo middleware
			reqInfo := RequestInfoFromContext(r.Context())

			// For top-level list of workspaces/namespaces, allow all authenticated users
			// but inject access filter so they only see resources they have bindings for.
			if verb == "list" && reqInfo.Scope == "platform" {
				if resourceChain == "workspaces" || resourceChain == "namespaces" {
					ctx := r.Context()
					af := &AccessFilter{}
					if resourceChain == "workspaces" {
						ids, ferr := checker.GetAccessibleWorkspaceIDs(ctx, userID)
						if ferr != nil {
							forbiddenError(w, "failed to check accessible workspaces")
							return
						}
						af.WorkspaceIDs = ids
					} else {
						ids, ferr := checker.GetAccessibleNamespaceIDs(ctx, userID)
						if ferr != nil {
							forbiddenError(w, "failed to check accessible namespaces")
							return
						}
						af.NamespaceIDs = ids
					}
					next.ServeHTTP(w, r.WithContext(WithAccessFilter(ctx, af)))
					return
				}
			}

			// Check permission
			allowed, err := checker.CheckPermission(r.Context(), userID, permCode, reqInfo.Scope, reqInfo.WorkspaceID, reqInfo.NamespaceID)
			if err != nil {
				forbiddenError(w, "failed to check permissions")
				return
			}
			if !allowed {
				forbiddenError(w, fmt.Sprintf("access denied: requires %s", permCode))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResolveResourceAndVerb extracts module, resource chain, and verb from an API URL path.
//
// URL format: /api/{module}/{version}/{resource}[/{id}][/{subresource}[/{id}]]...
//
// Examples:
//
//	GET  /api/iam/v1/users              → "iam", "users", "list"
//	GET  /api/iam/v1/users/1            → "iam", "users", "get"
//	POST /api/iam/v1/users              → "iam", "users", "create"
//	PUT  /api/iam/v1/users/1            → "iam", "users", "update"
//	PATCH /api/iam/v1/users/1           → "iam", "users", "patch"
//	DELETE /api/iam/v1/users/1          → "iam", "users", "delete"
//	DELETE /api/iam/v1/users            → "iam", "users", "deleteCollection"
//	POST /api/iam/v1/users/1/change-password → "iam", "users", "change-password"
//	GET  /api/iam/v1/users/1:workspaces → "iam", "users", "get"
//	GET  /api/iam/v1/workspaces/1/namespaces       → "iam", "workspaces:namespaces", "list"
//	GET  /api/iam/v1/workspaces/1/namespaces/2     → "iam", "workspaces:namespaces", "get"
//	POST /api/iam/v1/workspaces/1/namespaces/2/users → "iam", "workspaces:namespaces:users", "create"
func ResolveResourceAndVerb(method, path string) (module, resourceChain, verb string) {
	// Must start with /api/
	if !strings.HasPrefix(path, "/api/") {
		return "", "", ""
	}
	path = path[len("/api/"):]
	segments := strings.Split(path, "/")

	// Need at least module/version/resource (3 segments)
	if len(segments) < 3 {
		return "", "", ""
	}

	module = segments[0]  // e.g. "iam"
	// segments[1] is version, skip
	resourceSegments := segments[2:] // everything after module/version

	// Parse resource segments: alternating resource names and IDs
	// e.g. ["workspaces", "1", "namespaces", "2", "users"]
	var resources []string
	var lastIsID bool
	var actionName string

	for i, seg := range resourceSegments {
		// Check for custom verb (e.g. "1:workspaces")
		if colonIdx := strings.IndexByte(seg, ':'); colonIdx > 0 {
			// This is an ID with custom verb: "1:workspaces"
			// The custom verb maps to "get" on the parent resource
			lastIsID = true
			break
		}

		// Check if this looks like an ID (numeric)
		if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
			lastIsID = true
			continue
		}

		// Check if this is an action (last segment after an ID, and POST method)
		if lastIsID && i == len(resourceSegments)-1 && method == http.MethodPost {
			// Could be a sub-resource or an action
			// Actions typically contain hyphens (e.g. "change-password")
			if strings.Contains(seg, "-") {
				actionName = seg
				break
			}
		}

		resources = append(resources, seg)
		lastIsID = false
	}

	if len(resources) == 0 {
		return "", "", ""
	}

	resourceChain = strings.Join(resources, ":")

	// Determine verb
	if actionName != "" {
		verb = actionName
	} else {
		verb = httpMethodToVerb(method, lastIsID)
	}

	return module, resourceChain, verb
}

// httpMethodToVerb converts HTTP method + hasID to a REST verb.
func httpMethodToVerb(method string, hasID bool) string {
	switch method {
	case http.MethodGet:
		if hasID {
			return "get"
		}
		return "list"
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		if hasID {
			return "delete"
		}
		return "deleteCollection"
	default:
		return ""
	}
}

// isSelfUserQuery checks if the request is a user accessing their own user resource.
// This covers: GET /users/{id}, GET /users/{id}:workspaces, POST /users/{id}/change-password, etc.
func isSelfUserQuery(path, permCode string, currentUserID int64) bool {
	if permCode != "iam:users:get" && permCode != "iam:users:change-password" {
		return false
	}
	return extractUserIDFromPath(path) == currentUserID
}

// extractUserIDFromPath extracts the user ID from a /api/iam/v1/users/{id}[...] path.
func extractUserIDFromPath(path string) int64 {
	path = strings.TrimPrefix(path, "/api/")
	segments := strings.Split(path, "/")

	// Expect: module/version/users/{id}...
	if len(segments) < 4 || segments[2] != "users" {
		return 0
	}

	idStr := segments[3]
	// Handle custom verb: "123:workspaces" → "123"
	if idx := strings.IndexByte(idStr, ':'); idx > 0 {
		idStr = idStr[:idx]
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func forbiddenError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","message":"%s"}`, message)
}
