package iam

import (
	"context"
	"testing"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// --- TestNamespaceStorage_Get ---

func TestNamespaceStorage_Get(t *testing.T) {
	nsWithOwner := testNamespaceWithOwner(1, "my-namespace", 10, 100, "alice", "my-workspace")
	nsWithOwner.MemberCount = 5

	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			if id != 1 {
				t.Fatalf("expected id 1, got %d", id)
			}
			return nsWithOwner, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	obj, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"namespaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ns, ok := obj.(*Namespace)
	if !ok {
		t.Fatalf("expected *Namespace, got %T", obj)
	}

	if ns.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", ns.ObjectMeta.ID)
	}
	if ns.ObjectMeta.Name != "my-namespace" {
		t.Errorf("expected Name 'my-namespace', got %q", ns.ObjectMeta.Name)
	}
	if ns.TypeMeta.Kind != "Namespace" {
		t.Errorf("expected Kind 'Namespace', got %q", ns.TypeMeta.Kind)
	}
	if ns.Spec.OwnerID != "100" {
		t.Errorf("expected OwnerID '100', got %q", ns.Spec.OwnerID)
	}
	if ns.Spec.OwnerName != "alice" {
		t.Errorf("expected OwnerName 'alice', got %q", ns.Spec.OwnerName)
	}
	if ns.Spec.WorkspaceName != "my-workspace" {
		t.Errorf("expected WorkspaceName 'my-workspace', got %q", ns.Spec.WorkspaceName)
	}
	if ns.Spec.MemberCount != 5 {
		t.Errorf("expected MemberCount 5, got %d", ns.Spec.MemberCount)
	}
	if ns.Spec.WorkspaceID != "10" {
		t.Errorf("expected WorkspaceID '10', got %q", ns.Spec.WorkspaceID)
	}
	if ns.Spec.Status != "active" {
		t.Errorf("expected Status 'active', got %q", ns.Spec.Status)
	}
}

// --- TestNamespaceStorage_Get_InvalidID ---

func TestNamespaceStorage_Get_InvalidID(t *testing.T) {
	storage := NewNamespaceStorage(&mockNamespaceStore{}, nil, nil, nil)

	_, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"namespaceId": "abc"},
	})
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceStorage_List ---

func TestNamespaceStorage_List(t *testing.T) {
	nsStore := &mockNamespaceStore{
		ListFn: func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error) {
			return &db.ListResult[DBNamespaceWithOwner]{
				Items: []DBNamespaceWithOwner{
					{
						Namespace: generated.Namespace{
							ID:          1,
							Name:        "ns-one",
							DisplayName: "Namespace One",
							WorkspaceID: 10,
							OwnerID:     100,
							Visibility:  "private",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						OwnerUsername: "alice",
						WorkspaceName: "ws-one",
						MemberCount:   3,
					},
					{
						Namespace: generated.Namespace{
							ID:          2,
							Name:        "ns-two",
							DisplayName: "Namespace Two",
							WorkspaceID: 20,
							OwnerID:     200,
							Visibility:  "public",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						OwnerUsername: "bob",
						WorkspaceName: "ws-two",
						MemberCount:   1,
					},
				},
				TotalCount: 2,
			}, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	obj, err := storage.List(context.Background(), &rest.ListOptions{
		Filters: map[string]string{"status": "active"},
		Pagination: rest.Pagination{
			Page:     1,
			PageSize: 20,
		},
		SortBy:    "name",
		SortOrder: rest.SortOrderAsc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nsList, ok := obj.(*NamespaceList)
	if !ok {
		t.Fatalf("expected *NamespaceList, got %T", obj)
	}

	if nsList.TypeMeta.Kind != "NamespaceList" {
		t.Errorf("expected Kind 'NamespaceList', got %q", nsList.TypeMeta.Kind)
	}
	if nsList.TotalCount != 2 {
		t.Errorf("expected TotalCount 2, got %d", nsList.TotalCount)
	}
	if len(nsList.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(nsList.Items))
	}

	// Verify first namespace
	if nsList.Items[0].ObjectMeta.ID != "1" {
		t.Errorf("expected first item ID '1', got %q", nsList.Items[0].ObjectMeta.ID)
	}
	if nsList.Items[0].ObjectMeta.Name != "ns-one" {
		t.Errorf("expected first item Name 'ns-one', got %q", nsList.Items[0].ObjectMeta.Name)
	}
	if nsList.Items[0].Spec.OwnerName != "alice" {
		t.Errorf("expected first item OwnerName 'alice', got %q", nsList.Items[0].Spec.OwnerName)
	}
	if nsList.Items[0].Spec.WorkspaceName != "ws-one" {
		t.Errorf("expected first item WorkspaceName 'ws-one', got %q", nsList.Items[0].Spec.WorkspaceName)
	}
	if nsList.Items[0].Spec.MemberCount != 3 {
		t.Errorf("expected first item MemberCount 3, got %d", nsList.Items[0].Spec.MemberCount)
	}

	// Verify second namespace
	if nsList.Items[1].ObjectMeta.ID != "2" {
		t.Errorf("expected second item ID '2', got %q", nsList.Items[1].ObjectMeta.ID)
	}
	if nsList.Items[1].Spec.OwnerName != "bob" {
		t.Errorf("expected second item OwnerName 'bob', got %q", nsList.Items[1].Spec.OwnerName)
	}
	if nsList.Items[1].Spec.WorkspaceName != "ws-two" {
		t.Errorf("expected second item WorkspaceName 'ws-two', got %q", nsList.Items[1].Spec.WorkspaceName)
	}
}

// --- TestNamespaceStorage_List_FilterByWorkspace ---

func TestNamespaceStorage_List_FilterByWorkspace(t *testing.T) {
	var capturedQuery db.ListQuery

	nsStore := &mockNamespaceStore{
		ListFn: func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error) {
			capturedQuery = query
			return &db.ListResult[DBNamespaceWithOwner]{
				Items:      []DBNamespaceWithOwner{},
				TotalCount: 0,
			}, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	_, err := storage.List(context.Background(), &rest.ListOptions{
		PathParams: map[string]string{"workspaceId": "42"},
		Filters:    map[string]string{},
		Pagination: rest.Pagination{
			Page:     1,
			PageSize: 20,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wsFilter, ok := capturedQuery.Filters["workspace_id"]
	if !ok {
		t.Fatal("expected workspace_id filter to be added when pathParams has workspaceId")
	}
	if wsFilter != "42" {
		t.Errorf("expected workspace_id filter '42', got %q", wsFilter)
	}
}

// --- TestNamespaceStorage_Create ---

func TestNamespaceStorage_Create(t *testing.T) {
	var nsCreateCalled, wsGetByIDCalled, userGetByIDCalled bool

	nsStore := &mockNamespaceStore{
		CreateFn: func(ctx context.Context, ns *DBNamespace) (*DBNamespaceWithOwner, error) {
			nsCreateCalled = true
			if ns.Name != "new-namespace" {
				t.Errorf("expected name 'new-namespace', got %q", ns.Name)
			}
			if ns.DisplayName != "New Namespace" {
				t.Errorf("expected displayName 'New Namespace', got %q", ns.DisplayName)
			}
			if ns.WorkspaceID != 10 {
				t.Errorf("expected workspaceID 10, got %d", ns.WorkspaceID)
			}
			if ns.OwnerID != 100 {
				t.Errorf("expected ownerID 100, got %d", ns.OwnerID)
			}
			if ns.Visibility != "private" {
				t.Errorf("expected visibility 'private', got %q", ns.Visibility)
			}
			if ns.Status != "active" {
				t.Errorf("expected status 'active', got %q", ns.Status)
			}
			return testNamespaceWithOwner(1, "new-namespace", 10, 100, "alice", "my-workspace"), nil
		},
	}

	wsStore := &mockWorkspaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error) {
			wsGetByIDCalled = true
			if id != 10 {
				t.Errorf("expected workspace id 10, got %d", id)
			}
			return testWorkspaceWithOwner(10, "my-workspace", 100, "alice"), nil
		},
	}

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			userGetByIDCalled = true
			if id != 100 {
				t.Errorf("expected owner id 100, got %d", id)
			}
			return testUser(100, "alice", "alice@example.com"), nil
		},
	}

	storage := NewNamespaceStorage(nsStore, wsStore, userStore, nil)

	inputNs := &Namespace{
		Spec: NamespaceSpec{
			DisplayName: "New Namespace",
			Description: "A test namespace",
			WorkspaceID: "10",
			OwnerID:     "100",
			Visibility:  "private",
			Status:      "active",
		},
	}
	inputNs.ObjectMeta.Name = "new-namespace"

	obj, err := storage.Create(context.Background(), inputNs, &rest.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !wsGetByIDCalled {
		t.Error("expected wsStore.GetByID to be called to verify workspace exists")
	}
	if !userGetByIDCalled {
		t.Error("expected userStore.GetByID to be called to verify owner exists")
	}
	if !nsCreateCalled {
		t.Error("expected nsStore.Create to be called")
	}

	result, ok := obj.(*Namespace)
	if !ok {
		t.Fatalf("expected *Namespace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.ObjectMeta.Name != "new-namespace" {
		t.Errorf("expected Name 'new-namespace', got %q", result.ObjectMeta.Name)
	}
	if result.TypeMeta.Kind != "Namespace" {
		t.Errorf("expected Kind 'Namespace', got %q", result.TypeMeta.Kind)
	}
	if result.Spec.OwnerName != "alice" {
		t.Errorf("expected OwnerName 'alice', got %q", result.Spec.OwnerName)
	}
	if result.Spec.WorkspaceName != "my-workspace" {
		t.Errorf("expected WorkspaceName 'my-workspace', got %q", result.Spec.WorkspaceName)
	}
	if result.Spec.WorkspaceID != "10" {
		t.Errorf("expected WorkspaceID '10', got %q", result.Spec.WorkspaceID)
	}
}

// --- TestNamespaceStorage_Create_WorkspaceNotFound ---

func TestNamespaceStorage_Create_WorkspaceNotFound(t *testing.T) {
	wsStore := &mockWorkspaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error) {
			return nil, apierrors.NewNotFound("workspace", "999")
		},
	}

	userStore := &mockUserStore{}
	nsStore := &mockNamespaceStore{}

	storage := NewNamespaceStorage(nsStore, wsStore, userStore, nil)

	inputNs := &Namespace{
		Spec: NamespaceSpec{
			DisplayName: "Test Namespace",
			WorkspaceID: "999",
			OwnerID:     "100",
			Status:      "active",
		},
	}
	inputNs.ObjectMeta.Name = "test-namespace"

	_, err := storage.Create(context.Background(), inputNs, &rest.CreateOptions{})
	if err == nil {
		t.Fatal("expected error when workspace not found, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceStorage_Update ---

func TestNamespaceStorage_Update(t *testing.T) {
	existingNsWithOwner := testNamespaceWithOwner(1, "my-namespace", 10, 100, "alice", "my-workspace")

	updatedDBNs := testNamespaceWithOwner(1, "updated-namespace", 10, 100, "alice", "my-workspace")
	updatedDBNs.DisplayName = "Updated Namespace"
	updatedDBNs.Description = "Updated description"

	var getByIDCalled, updateCalled bool

	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			getByIDCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			return existingNsWithOwner, nil
		},
		UpdateFn: func(ctx context.Context, ns *DBNamespace) (*DBNamespaceWithOwner, error) {
			updateCalled = true
			if ns.ID != 1 {
				t.Errorf("expected namespace ID 1, got %d", ns.ID)
			}
			if ns.DisplayName != "Updated Namespace" {
				t.Errorf("expected displayName 'Updated Namespace', got %q", ns.DisplayName)
			}
			if ns.OwnerID != 100 {
				t.Errorf("expected ownerID 100, got %d", ns.OwnerID)
			}
			// WorkspaceID should be preserved from existing
			if ns.WorkspaceID != 10 {
				t.Errorf("expected workspaceID 10, got %d", ns.WorkspaceID)
			}
			return updatedDBNs, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	inputNs := &Namespace{
		Spec: NamespaceSpec{
			DisplayName: "Updated Namespace",
			Description: "Updated description",
			OwnerID:     "100",
			Visibility:  "private",
			Status:      "active",
		},
	}
	inputNs.ObjectMeta.Name = "updated-namespace"

	obj, err := storage.Update(context.Background(), inputNs, &rest.UpdateOptions{
		PathParams: map[string]string{"namespaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !getByIDCalled {
		t.Error("expected nsStore.GetByID to be called to get existing workspace_id")
	}
	if !updateCalled {
		t.Error("expected nsStore.Update to be called")
	}

	result, ok := obj.(*Namespace)
	if !ok {
		t.Fatalf("expected *Namespace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.DisplayName != "Updated Namespace" {
		t.Errorf("expected DisplayName 'Updated Namespace', got %q", result.Spec.DisplayName)
	}
	if result.Spec.Description != "Updated description" {
		t.Errorf("expected Description 'Updated description', got %q", result.Spec.Description)
	}
	if result.TypeMeta.Kind != "Namespace" {
		t.Errorf("expected Kind 'Namespace', got %q", result.TypeMeta.Kind)
	}
}

// --- TestNamespaceStorage_Patch ---

func TestNamespaceStorage_Patch(t *testing.T) {
	patchedDBNs := testNamespaceWithOwner(1, "my-namespace", 10, 100, "alice", "my-workspace")
	patchedDBNs.DisplayName = "Patched Namespace"
	patchedDBNs.Description = "Original description"

	var patchCalled bool

	nsStore := &mockNamespaceStore{
		PatchFn: func(ctx context.Context, id int64, ns *DBNamespace) (*DBNamespaceWithOwner, error) {
			patchCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			if ns.DisplayName != "Patched Namespace" {
				t.Errorf("expected displayName 'Patched Namespace', got %q", ns.DisplayName)
			}
			// Description should be empty (zero value) since patch input did not set it
			if ns.Description != "" {
				t.Errorf("expected description '' (zero value), got %q", ns.Description)
			}
			// WorkspaceID should be 0 (zero value) since patch input did not set it
			if ns.WorkspaceID != 0 {
				t.Errorf("expected workspaceID 0 (zero value), got %d", ns.WorkspaceID)
			}
			return patchedDBNs, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	// Patch only the displayName
	inputNs := &Namespace{
		Spec: NamespaceSpec{
			DisplayName: "Patched Namespace",
		},
	}

	obj, err := storage.Patch(context.Background(), inputNs, &rest.PatchOptions{
		PathParams: map[string]string{"namespaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !patchCalled {
		t.Error("expected nsStore.Patch to be called")
	}

	result, ok := obj.(*Namespace)
	if !ok {
		t.Fatalf("expected *Namespace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.DisplayName != "Patched Namespace" {
		t.Errorf("expected DisplayName 'Patched Namespace', got %q", result.Spec.DisplayName)
	}
}

// --- TestNamespaceStorage_Delete ---

func TestNamespaceStorage_Delete(t *testing.T) {
	deleteCalled := false

	nsStore := &mockNamespaceStore{
		DeleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			return nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	err := storage.Delete(context.Background(), &rest.DeleteOptions{
		PathParams: map[string]string{"namespaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("expected Delete to be called on the store")
	}
}

// --- TestNamespaceStorage_DeleteCollection ---

func TestNamespaceStorage_DeleteCollection(t *testing.T) {
	nsStore := &mockNamespaceStore{
		DeleteByIDsFn: func(ctx context.Context, ids []int64) (int64, error) {
			if len(ids) != 3 {
				t.Errorf("expected 3 IDs, got %d", len(ids))
			}
			expectedIDs := []int64{1, 2, 3}
			for i, id := range ids {
				if id != expectedIDs[i] {
					t.Errorf("expected ID %d at index %d, got %d", expectedIDs[i], i, id)
				}
			}
			// Simulate that 2 out of 3 were actually deleted
			return 2, nil
		},
	}

	storage := NewNamespaceStorage(nsStore, nil, nil, nil)

	result, err := storage.DeleteCollection(context.Background(), []string{"1", "2", "3"}, &rest.DeleteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SuccessCount != 2 {
		t.Errorf("expected SuccessCount 2, got %d", result.SuccessCount)
	}
	if result.FailedCount != 1 {
		t.Errorf("expected FailedCount 1, got %d", result.FailedCount)
	}
}
