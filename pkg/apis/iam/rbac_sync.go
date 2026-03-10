package iam

import (
	"context"
	"fmt"
	"strings"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
)

// PermissionLookup maps module → resourceChain → verb → permCode.
// Example: "iam" → "workspaces:namespaces" → "list" → "iam:namespaces:list"
type PermissionLookup map[string]map[string]map[string]string

// Get returns the permission code for a given module, resourceChain, and verb.
func (l PermissionLookup) Get(module, resourceChain, verb string) string {
	if m, ok := l[module]; ok {
		if r, ok := m[resourceChain]; ok {
			return r[verb]
		}
	}
	return ""
}

// storageEntry represents a discovered storage endpoint with its resource path.
type storageEntry struct {
	storage   rest.Storage
	codeParts []string // e.g. ["namespaces", "users"]
	pathParts []string // e.g. ["workspaces", "namespaces", "users"]
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
			storage:   res.Storage,
			codeParts: codeParts,
			pathParts: pathParts,
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

// canonicalCode builds a permission code from module + canonical codeParts + verb.
// e.g. ("iam", ["namespaces", "users"], "list") → "iam:namespaces:users:list"
func canonicalCode(module string, codeParts []string, verb string) string {
	return module + ":" + strings.Join(codeParts, ":") + ":" + verb
}

// generatePermissions creates permission definitions from canonical entries.
func generatePermissions(canonical map[rest.Storage]storageEntry, module string, group *rest.APIGroupInfo, entries []storageEntry) []permissionDef {
	var perms []permissionDef

	// Also collect actions from the original resource tree
	actionPerms := collectActions(group.Resources, nil, module, group, canonical, entries)
	perms = append(perms, actionPerms...)

	for storage, entry := range canonical {
		basePath := buildAPIPath(group, entry.pathParts)
		verbs := detectVerbs(storage)
		scope := scopeForStorage(storage, entries)

		for _, verb := range verbs {
			method := verbMethods[verb]
			code := canonicalCode(module, entry.codeParts, verb)
			path := verbPath(basePath, verb, entry.pathParts)
			perms = append(perms, permissionDef{
				code:   code,
				method: method,
				path:   path,
				scope:  scope,
			})
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
			scope := scopeForStorage(res.Storage, entries)
			code := module + ":" + strings.Join(canonEntry.codeParts, ":") + ":" + action.Name
			basePath := buildAPIPath(group, pathParts)
			path := basePath + "/{" + idParam(res.Name) + "}/" + action.Name
			perms = append(perms, permissionDef{
				code:   code,
				method: action.Method,
				path:   path,
				scope:  scope,
			})
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
		lookup[module] = make(map[string]map[string]string)
	}

	for _, entry := range entries {
		canonEntry := canonical[entry.storage]
		verbs := detectVerbs(entry.storage)

		// Resource chain key: join pathParts with ":"
		chainKey := strings.Join(entry.pathParts, ":")

		if lookup[module][chainKey] == nil {
			lookup[module][chainKey] = make(map[string]string)
		}
		for _, verb := range verbs {
			lookup[module][chainKey][verb] = canonicalCode(module, canonEntry.codeParts, verb)
		}

		// Also register by the canonical codeParts chain
		canonChainKey := strings.Join(canonEntry.codeParts, ":")
		if lookup[module][canonChainKey] == nil {
			lookup[module][canonChainKey] = make(map[string]string)
		}
		for _, verb := range verbs {
			lookup[module][canonChainKey][verb] = canonicalCode(module, canonEntry.codeParts, verb)
		}
	}
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
