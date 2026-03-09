package iam

import (
	"context"
	"sync"
	"time"

	"lcp.io/lcp/lib/rest/filters"
)

// namespaceResolver caches namespace→workspace mappings to avoid
// per-request DB lookups in WithRequestInfo middleware.
type namespaceResolver struct {
	mu      sync.RWMutex
	cache   map[int64]resolverEntry
	ttl     time.Duration
	nsStore NamespaceStore
}

type resolverEntry struct {
	workspaceID int64
	expiresAt   time.Time
}

// NewNamespaceResolver creates a NamespaceResolver backed by NamespaceStore with TTL caching.
func NewNamespaceResolver(nsStore NamespaceStore, ttl time.Duration) filters.NamespaceResolver {
	return &namespaceResolver{
		cache:   make(map[int64]resolverEntry),
		ttl:     ttl,
		nsStore: nsStore,
	}
}

func (r *namespaceResolver) GetWorkspaceID(namespaceID int64) (int64, bool) {
	r.mu.RLock()
	if entry, ok := r.cache[namespaceID]; ok && time.Now().Before(entry.expiresAt) {
		r.mu.RUnlock()
		return entry.workspaceID, true
	}
	r.mu.RUnlock()

	ns, err := r.nsStore.GetByID(context.Background(), namespaceID)
	if err != nil {
		return 0, false
	}

	r.mu.Lock()
	r.cache[namespaceID] = resolverEntry{
		workspaceID: ns.WorkspaceID,
		expiresAt:   time.Now().Add(r.ttl),
	}
	r.mu.Unlock()

	return ns.WorkspaceID, true
}

// InvalidateNamespace removes a cached namespace entry (e.g. on delete or workspace transfer).
func (r *namespaceResolver) InvalidateNamespace(namespaceID int64) {
	r.mu.Lock()
	delete(r.cache, namespaceID)
	r.mu.Unlock()
}
