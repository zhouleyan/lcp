package iam

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"lcp.io/lcp/pkg/db"
)

// mockNamespaceStoreForResolver implements the subset of NamespaceStore needed by resolver.
type mockNamespaceStoreForResolver struct {
	data     map[int64]*DBNamespaceWithOwner
	callCount atomic.Int64
}

func (m *mockNamespaceStoreForResolver) GetByID(_ context.Context, id int64) (*DBNamespaceWithOwner, error) {
	m.callCount.Add(1)
	ns, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return ns, nil
}

// Unused methods to satisfy NamespaceStore interface.
func (m *mockNamespaceStoreForResolver) Create(_ context.Context, _ *DBNamespace) (*DBNamespaceWithOwner, error) {
	return nil, nil
}
func (m *mockNamespaceStoreForResolver) GetByName(_ context.Context, _ string) (*DBNamespace, error) {
	return nil, nil
}
func (m *mockNamespaceStoreForResolver) Update(_ context.Context, _ *DBNamespace) (*DBNamespace, error) {
	return nil, nil
}
func (m *mockNamespaceStoreForResolver) Patch(_ context.Context, _ int64, _ *DBNamespace) (*DBNamespace, error) {
	return nil, nil
}
func (m *mockNamespaceStoreForResolver) Delete(_ context.Context, _ int64) error { return nil }
func (m *mockNamespaceStoreForResolver) DeleteByIDs(_ context.Context, _ []int64) (int64, error) {
	return 0, nil
}
func (m *mockNamespaceStoreForResolver) List(_ context.Context, _ db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error) {
	return nil, nil
}
func (m *mockNamespaceStoreForResolver) CountUsers(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func newMockNSStore() *mockNamespaceStoreForResolver {
	return &mockNamespaceStoreForResolver{
		data: map[int64]*DBNamespaceWithOwner{
			456: {Namespace: DBNamespace{ID: 456, WorkspaceID: 789}},
			100: {Namespace: DBNamespace{ID: 100, WorkspaceID: 200}},
		},
	}
}

func TestNamespaceResolver_GetWorkspaceID(t *testing.T) {
	store := newMockNSStore()
	resolver := NewNamespaceResolver(store, 1*time.Minute)

	// First call: cache miss, queries DB
	wsID, ok := resolver.GetWorkspaceID(context.Background(),456)
	if !ok || wsID != 789 {
		t.Errorf("got wsID=%d ok=%v, want 789/true", wsID, ok)
	}
	if store.callCount.Load() != 1 {
		t.Errorf("expected 1 DB call, got %d", store.callCount.Load())
	}

	// Second call: cache hit, no additional DB call
	wsID, ok = resolver.GetWorkspaceID(context.Background(),456)
	if !ok || wsID != 789 {
		t.Errorf("got wsID=%d ok=%v, want 789/true", wsID, ok)
	}
	if store.callCount.Load() != 1 {
		t.Errorf("expected still 1 DB call, got %d", store.callCount.Load())
	}
}

func TestNamespaceResolver_NotFound(t *testing.T) {
	store := newMockNSStore()
	resolver := NewNamespaceResolver(store, 1*time.Minute)

	_, ok := resolver.GetWorkspaceID(context.Background(),999)
	if ok {
		t.Error("expected not found")
	}
}

func TestNamespaceResolver_TTLExpiry(t *testing.T) {
	store := newMockNSStore()
	resolver := NewNamespaceResolver(store, 1*time.Millisecond)

	resolver.GetWorkspaceID(context.Background(),456)
	time.Sleep(5 * time.Millisecond)

	// Cache expired, should query DB again
	resolver.GetWorkspaceID(context.Background(),456)
	if store.callCount.Load() != 2 {
		t.Errorf("expected 2 DB calls after TTL expiry, got %d", store.callCount.Load())
	}
}

func TestNamespaceResolver_Invalidate(t *testing.T) {
	store := newMockNSStore()
	r := NewNamespaceResolver(store, 1*time.Minute)

	r.GetWorkspaceID(context.Background(),456)

	// Invalidate
	r.(*namespaceResolver).InvalidateNamespace(456)

	// Next call should query DB again
	r.GetWorkspaceID(context.Background(),456)
	if store.callCount.Load() != 2 {
		t.Errorf("expected 2 DB calls after invalidation, got %d", store.callCount.Load())
	}
}
