package iam

import (
	"context"
	"fmt"
	"strings"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
)

// PermissionLookup maps module → resourceChain → verb → []permCode.
// For normal resources, the slice contains a single auto-derived code.
// For resources with PermissionTargets, it contains the configured targets.
type PermissionLookup map[string]map[string]map[string][]string

// Get returns the permission codes for a given module, resourceChain, and verb.
func (l PermissionLookup) Get(module, resourceChain, verb string) []string {
	if m, ok := l[module]; ok {
		if r, ok := m[resourceChain]; ok {
			return r[verb]
		}
	}
	return nil
}

// storageEntry represents a discovered storage endpoint with its resource path.
type storageEntry struct {
	storage           rest.Storage
	codeParts         []string // e.g. ["namespaces", "users"]
	pathParts         []string // e.g. ["workspaces", "namespaces", "users"]
	permissionTargets []string // from ResourceInfo.PermissionTargets; overrides auto-derived codes
}

// permissionDef is a generated permission definition.
type permissionDef struct {
	code        string
	method      string
	path        string
	scope       string
	description string
}

// verbMethods maps REST verbs to HTTP methods.
var verbMethods = map[string]string{
	"list":             "GET",
	"get":              "GET",
	"create":           "POST",
	"update":           "PUT",
	"patch":            "PATCH",
	"delete":           "DELETE",
	"deleteCollection": "DELETE",
}

// SyncPermissions scans all API groups, generates permission codes, syncs to DB,
// and returns a PermissionLookup for runtime use.
func SyncPermissions(ctx context.Context, permStore PermissionStore, apiGroups []*rest.APIGroupInfo) (PermissionLookup, error) {
	lookup := make(PermissionLookup)

	for _, group := range apiGroups {
		module := group.GroupName
		if module == "" {
			module = "core"
		}

		// 1. Collect all storage entries from the resource tree
		entries := collectStorageEntries(group.Resources, nil, nil)

		// 2. Canonicalize: pick shortest codeParts per storage pointer
		canonical := canonicalize(entries)

		// 3. Generate permission definitions from canonical entries
		perms := generatePermissions(canonical, module, group, entries)

		// 4. Build lookup table (all entries map to canonical codes)
		buildLookup(lookup, entries, canonical, module)

		// 5. Sync to DB
		if err := syncPermissionsToDB(ctx, permStore, module, perms); err != nil {
			return nil, fmt.Errorf("sync permissions for module %s: %w", module, err)
		}

		logger.Infof("synced %d permissions for module %q", len(perms), module)
	}

	return lookup, nil
}

// BuildPermissionLookup derives a PermissionLookup from API group definitions.
// Pure function — no DB access, no side effects.
func BuildPermissionLookup(apiGroups []*rest.APIGroupInfo) PermissionLookup {
	lookup := make(PermissionLookup)
	for _, group := range apiGroups {
		module := group.GroupName
		if module == "" {
			module = "core"
		}
		entries := collectStorageEntries(group.Resources, nil, nil)
		canonical := canonicalize(entries)
		buildLookup(lookup, entries, canonical, module)
	}
	return lookup
}

// collectStorageEntries recursively walks the resource tree and collects storage entries.
func collectStorageEntries(resources []rest.ResourceInfo, parentCodeParts, parentPathParts []string) []storageEntry {
	var entries []storageEntry

	for _, res := range resources {
		pathParts := append(append([]string{}, parentPathParts...), res.Name)

		// For the canonical code, we use the resource name itself (not the full nested path)
		codeParts := append(append([]string{}, parentCodeParts...), res.Name)

		entries = append(entries, storageEntry{
			storage:           res.Storage,
			codeParts:         codeParts,
			pathParts:         pathParts,
			permissionTargets: res.PermissionTargets,
		})

		// Recurse into sub-resources
		if len(res.SubResources) > 0 {
			entries = append(entries, collectStorageEntries(res.SubResources, codeParts, pathParts)...)
		}
	}

	return entries
}

// canonicalize groups entries by storage pointer and picks the shortest codeParts.
func canonicalize(entries []storageEntry) map[rest.Storage]storageEntry {
	canonical := make(map[rest.Storage]storageEntry)
	for _, e := range entries {
		if existing, ok := canonical[e.storage]; !ok || len(e.codeParts) < len(existing.codeParts) {
			canonical[e.storage] = e
		}
	}
	return canonical
}

// scopeForStorage determines the permission scope based on the deepest nesting
// in the resource tree. Top-level resources are "platform", resources nested
// under "workspaces" are "workspace", and resources nested under
// "workspaces" > "namespaces" are "namespace".
func scopeForStorage(storage rest.Storage, entries []storageEntry) string {
	var deepest []string
	for _, e := range entries {
		if e.storage == storage && len(e.pathParts) > len(deepest) {
			deepest = e.pathParts
		}
	}
	wsIdx := -1
	nsIdx := -1
	for i, part := range deepest {
		if part == "workspaces" && wsIdx == -1 {
			wsIdx = i
		}
		if part == "namespaces" && wsIdx >= 0 {
			nsIdx = i
		}
	}
	if nsIdx > wsIdx && wsIdx >= 0 {
		return "namespace"
	}
	if wsIdx >= 0 {
		return "workspace"
	}
	return "platform"
}

// scopeSegments are resource names that represent scope hierarchy, not actual resources
// in the permission code. They are stripped from codeParts to produce simplified codes.
var scopeSegments = map[string]bool{"workspaces": true, "namespaces": true}

// stripScopeSegments removes "workspaces" and "namespaces" from codeParts when
// they appear as parent scope markers, but keeps them if they are the leaf resource.
// e.g. ["workspaces", "namespaces", "users"] → ["users"]
// e.g. ["workspaces", "users"] → ["users"]
// e.g. ["workspaces"] → ["workspaces"]  (leaf resource, kept)
// e.g. ["namespaces"] → ["namespaces"]  (leaf resource, kept)
func stripScopeSegments(codeParts []string) []string {
	if len(codeParts) <= 1 {
		return codeParts
	}
	var result []string
	for i, p := range codeParts {
		// Only strip scope segments that are NOT the last element
		if scopeSegments[p] && i < len(codeParts)-1 {
			continue
		}
		result = append(result, p)
	}
	return result
}

// canonicalCode builds a permission code from module + canonical codeParts + verb.
// Scope segments (workspaces/namespaces) are stripped from codeParts.
// e.g. ("iam", ["namespaces", "users"], "list") → "iam:users:list"
func canonicalCode(module string, codeParts []string, verb string) string {
	stripped := stripScopeSegments(codeParts)
	return module + ":" + strings.Join(stripped, ":") + ":" + verb
}

// scopesUpTo returns all scopes from the given natural scope up to platform.
// e.g. "namespace" → ["namespace", "workspace", "platform"]
//
//	"workspace" → ["workspace", "platform"]
//	"platform"  → ["platform"]
func scopesUpTo(naturalScope string) []string {
	switch naturalScope {
	case "namespace":
		return []string{"namespace", "workspace", "platform"}
	case "workspace":
		return []string{"workspace", "platform"}
	default:
		return []string{"platform"}
	}
}

// generatePermissions creates permission definitions from canonical entries.
// Each permission code generates records from its natural scope UP to platform.
func generatePermissions(canonical map[rest.Storage]storageEntry, module string, group *rest.APIGroupInfo, entries []storageEntry) []permissionDef {
	var perms []permissionDef

	// Also collect actions from the original resource tree
	actionPerms := collectActions(group.Resources, nil, module, group, canonical, entries)
	perms = append(perms, actionPerms...)

	for storage, entry := range canonical {
		// Skip resources with PermissionTargets — they map to existing permissions, not new ones
		if hasPermissionTargets(storage, entries) {
			continue
		}

		basePath := buildAPIPath(group, entry.pathParts)
		verbs := detectVerbs(storage)
		naturalScope := scopeForStorage(storage, entries)

		for _, verb := range verbs {
			method := verbMethods[verb]
			code := canonicalCode(module, entry.codeParts, verb)
			path := verbPath(basePath, verb, entry.pathParts)

			for _, scope := range scopesUpTo(naturalScope) {
				perms = append(perms, permissionDef{
					code:   code,
					method: method,
					path:   path,
					scope:  scope,
				})
			}
		}
	}

	return perms
}

// collectActions walks the resource tree to find ActionInfo entries.
func collectActions(resources []rest.ResourceInfo, parentPathParts []string, module string, group *rest.APIGroupInfo, canonical map[rest.Storage]storageEntry, entries []storageEntry) []permissionDef {
	var perms []permissionDef

	for _, res := range resources {
		pathParts := append(append([]string{}, parentPathParts...), res.Name)

		for _, action := range res.Actions {
			// Use canonical codeParts for the resource to derive action code
			canonEntry := canonical[res.Storage]
			naturalScope := scopeForStorage(res.Storage, entries)
			stripped := stripScopeSegments(canonEntry.codeParts)
			code := module + ":" + strings.Join(stripped, ":") + ":" + action.Name
			basePath := buildAPIPath(group, pathParts)
			path := basePath + "/{" + idParam(res.Name) + "}/" + action.Name

			for _, scope := range scopesUpTo(naturalScope) {
				perms = append(perms, permissionDef{
					code:   code,
					method: action.Method,
					path:   path,
					scope:  scope,
				})
			}
		}

		if len(res.SubResources) > 0 {
			perms = append(perms, collectActions(res.SubResources, pathParts, module, group, canonical, entries)...)
		}
	}

	return perms
}

// detectVerbs checks which REST interfaces a storage implements.
func detectVerbs(s rest.Storage) []string {
	var verbs []string
	if _, ok := s.(rest.Lister); ok {
		verbs = append(verbs, "list")
	}
	if _, ok := s.(rest.Getter); ok {
		verbs = append(verbs, "get")
	}
	if _, ok := s.(rest.Creator); ok {
		verbs = append(verbs, "create")
	}
	if _, ok := s.(rest.Updater); ok {
		verbs = append(verbs, "update")
	}
	if _, ok := s.(rest.Patcher); ok {
		verbs = append(verbs, "patch")
	}
	if _, ok := s.(rest.Deleter); ok {
		verbs = append(verbs, "delete")
	}
	if _, ok := s.(rest.CollectionDeleter); ok {
		verbs = append(verbs, "deleteCollection")
	}
	return verbs
}

// buildAPIPath builds the URL path prefix for a resource.
func buildAPIPath(group *rest.APIGroupInfo, pathParts []string) string {
	base := group.BasePath()
	var parts []string
	for i, p := range pathParts {
		parts = append(parts, p)
		if i < len(pathParts)-1 {
			parts = append(parts, "{"+idParam(p)+"}")
		}
	}
	return base + "/" + strings.Join(parts, "/")
}

// verbPath returns the full path for a verb.
func verbPath(basePath, verb string, pathParts []string) string {
	resourceName := pathParts[len(pathParts)-1]
	switch verb {
	case "list", "create", "deleteCollection":
		return basePath
	default:
		return basePath + "/{" + idParam(resourceName) + "}"
	}
}

// idParam converts a plural resource name to its ID parameter name.
// e.g. "users" → "userId", "workspaces" → "workspaceId"
func idParam(name string) string {
	singular := strings.TrimSuffix(name, "s")
	return singular + "Id"
}

// buildLookup populates the PermissionLookup for all entries (including non-canonical aliases).
func buildLookup(lookup PermissionLookup, entries []storageEntry, canonical map[rest.Storage]storageEntry, module string) {
	if lookup[module] == nil {
		lookup[module] = make(map[string]map[string][]string)
	}

	for _, entry := range entries {
		verbs := detectVerbs(entry.storage)

		// Resource chain key: join pathParts with ":"
		chainKey := strings.Join(entry.pathParts, ":")

		if lookup[module][chainKey] == nil {
			lookup[module][chainKey] = make(map[string][]string)
		}

		// If the entry has PermissionTargets, use them directly instead of auto-derived codes
		if len(entry.permissionTargets) > 0 {
			for _, verb := range verbs {
				lookup[module][chainKey][verb] = entry.permissionTargets
			}
			continue
		}

		canonEntry := canonical[entry.storage]
		for _, verb := range verbs {
			code := canonicalCode(module, canonEntry.codeParts, verb)
			lookup[module][chainKey][verb] = []string{code}
		}

		// Also register by the canonical codeParts chain
		canonChainKey := strings.Join(canonEntry.codeParts, ":")
		if lookup[module][canonChainKey] == nil {
			lookup[module][canonChainKey] = make(map[string][]string)
		}
		for _, verb := range verbs {
			code := canonicalCode(module, canonEntry.codeParts, verb)
			lookup[module][canonChainKey][verb] = []string{code}
		}
	}
}

// hasPermissionTargets checks if any entry for the given storage has PermissionTargets set.
func hasPermissionTargets(storage rest.Storage, entries []storageEntry) bool {
	for _, e := range entries {
		if e.storage == storage && len(e.permissionTargets) > 0 {
			return true
		}
	}
	return false
}

// syncPermissionsToDB batch-upserts all permissions and cleans up stale ones in a single transaction.
func syncPermissionsToDB(ctx context.Context, permStore PermissionStore, module string, perms []permissionDef) error {
	dbPerms := make([]DBPermission, len(perms))
	for i, p := range perms {
		dbPerms[i] = DBPermission{
			Code:        p.code,
			Method:      p.method,
			Path:        p.path,
			Scope:       p.scope,
			Description: p.description,
		}
	}

	modulePrefix := module + ":"
	return permStore.SyncModule(ctx, modulePrefix, dbPerms)
}
