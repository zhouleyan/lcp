package iam

import (
	"context"
	"testing"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// --- TestWorkspaceStorage_Get ---

func TestWorkspaceStorage_Get(t *testing.T) {
	wsWithOwner := testWorkspaceWithOwner(1, "my-workspace", 10, "alice")
	wsWithOwner.NamespaceCount = 3
	wsWithOwner.MemberCount = 5

	wsStore := &mockWorkspaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error) {
			if id != 1 {
				t.Fatalf("expected id 1, got %d", id)
			}
			return wsWithOwner, nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, nil, nil)

	obj, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ws, ok := obj.(*Workspace)
	if !ok {
		t.Fatalf("expected *Workspace, got %T", obj)
	}

	if ws.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", ws.ObjectMeta.ID)
	}
	if ws.ObjectMeta.Name != "my-workspace" {
		t.Errorf("expected Name 'my-workspace', got %q", ws.ObjectMeta.Name)
	}
	if ws.TypeMeta.Kind != "Workspace" {
		t.Errorf("expected Kind 'Workspace', got %q", ws.TypeMeta.Kind)
	}
	if ws.Spec.OwnerID != "10" {
		t.Errorf("expected OwnerID '10', got %q", ws.Spec.OwnerID)
	}
	if ws.Spec.OwnerName != "alice" {
		t.Errorf("expected OwnerName 'alice', got %q", ws.Spec.OwnerName)
	}
	if ws.Spec.NamespaceCount != 3 {
		t.Errorf("expected NamespaceCount 3, got %d", ws.Spec.NamespaceCount)
	}
	if ws.Spec.MemberCount != 5 {
		t.Errorf("expected MemberCount 5, got %d", ws.Spec.MemberCount)
	}
	if ws.Spec.Status != "active" {
		t.Errorf("expected Status 'active', got %q", ws.Spec.Status)
	}
}

// --- TestWorkspaceStorage_Get_InvalidID ---

func TestWorkspaceStorage_Get_InvalidID(t *testing.T) {
	storage := NewWorkspaceStorage(&mockWorkspaceStore{}, nil, nil)

	_, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"workspaceId": "abc"},
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

// --- TestWorkspaceStorage_List ---

func TestWorkspaceStorage_List(t *testing.T) {
	wsStore := &mockWorkspaceStore{
		ListFn: func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwner], error) {
			return &db.ListResult[DBWorkspaceWithOwner]{
				Items: []DBWorkspaceWithOwner{
					{
						Workspace: generated.Workspace{
							ID:          1,
							Name:        "ws-one",
							DisplayName: "Workspace One",
							OwnerID:     10,
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						OwnerUsername:  "alice",
						NamespaceCount: 2,
						MemberCount:    3,
					},
					{
						Workspace: generated.Workspace{
							ID:          2,
							Name:        "ws-two",
							DisplayName: "Workspace Two",
							OwnerID:     20,
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						OwnerUsername:  "bob",
						NamespaceCount: 1,
						MemberCount:    1,
					},
				},
				TotalCount: 2,
			}, nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, nil, nil)

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

	wsList, ok := obj.(*WorkspaceList)
	if !ok {
		t.Fatalf("expected *WorkspaceList, got %T", obj)
	}

	if wsList.TypeMeta.Kind != "WorkspaceList" {
		t.Errorf("expected Kind 'WorkspaceList', got %q", wsList.TypeMeta.Kind)
	}
	if wsList.TotalCount != 2 {
		t.Errorf("expected TotalCount 2, got %d", wsList.TotalCount)
	}
	if len(wsList.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(wsList.Items))
	}

	// Verify first workspace
	if wsList.Items[0].ObjectMeta.ID != "1" {
		t.Errorf("expected first item ID '1', got %q", wsList.Items[0].ObjectMeta.ID)
	}
	if wsList.Items[0].ObjectMeta.Name != "ws-one" {
		t.Errorf("expected first item Name 'ws-one', got %q", wsList.Items[0].ObjectMeta.Name)
	}
	if wsList.Items[0].Spec.OwnerName != "alice" {
		t.Errorf("expected first item OwnerName 'alice', got %q", wsList.Items[0].Spec.OwnerName)
	}
	if wsList.Items[0].Spec.NamespaceCount != 2 {
		t.Errorf("expected first item NamespaceCount 2, got %d", wsList.Items[0].Spec.NamespaceCount)
	}
	if wsList.Items[0].Spec.MemberCount != 3 {
		t.Errorf("expected first item MemberCount 3, got %d", wsList.Items[0].Spec.MemberCount)
	}

	// Verify second workspace
	if wsList.Items[1].ObjectMeta.ID != "2" {
		t.Errorf("expected second item ID '2', got %q", wsList.Items[1].ObjectMeta.ID)
	}
	if wsList.Items[1].Spec.OwnerName != "bob" {
		t.Errorf("expected second item OwnerName 'bob', got %q", wsList.Items[1].Spec.OwnerName)
	}
}

// --- TestWorkspaceStorage_Create ---

func TestWorkspaceStorage_Create(t *testing.T) {
	var createCalled, userGetByIDCalled bool

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			userGetByIDCalled = true
			if id != 10 {
				t.Errorf("expected owner id 10, got %d", id)
			}
			return testUser(10, "alice", "alice@example.com"), nil
		},
	}

	wsStore := &mockWorkspaceStore{
		CreateFn: func(ctx context.Context, ws *DBWorkspace) (*DBWorkspaceWithOwner, error) {
			createCalled = true
			if ws.Name != "new-workspace" {
				t.Errorf("expected name 'new-workspace', got %q", ws.Name)
			}
			if ws.DisplayName != "New Workspace" {
				t.Errorf("expected displayName 'New Workspace', got %q", ws.DisplayName)
			}
			if ws.OwnerID != 10 {
				t.Errorf("expected ownerID 10, got %d", ws.OwnerID)
			}
			if ws.Status != "active" {
				t.Errorf("expected status 'active', got %q", ws.Status)
			}
			return testWorkspaceWithOwner(1, "new-workspace", 10, "alice"), nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, userStore, nil)

	inputWs := &Workspace{
		Spec: WorkspaceSpec{
			DisplayName: "New Workspace",
			Description: "A test workspace",
			OwnerID:     "10",
			Status:      "active",
		},
	}
	inputWs.ObjectMeta.Name = "new-workspace"

	obj, err := storage.Create(context.Background(), inputWs, &rest.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !userGetByIDCalled {
		t.Error("expected userStore.GetByID to be called to verify owner")
	}
	if !createCalled {
		t.Error("expected wsStore.Create to be called")
	}

	result, ok := obj.(*Workspace)
	if !ok {
		t.Fatalf("expected *Workspace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.ObjectMeta.Name != "new-workspace" {
		t.Errorf("expected Name 'new-workspace', got %q", result.ObjectMeta.Name)
	}
	if result.TypeMeta.Kind != "Workspace" {
		t.Errorf("expected Kind 'Workspace', got %q", result.TypeMeta.Kind)
	}
	if result.Spec.OwnerName != "alice" {
		t.Errorf("expected OwnerName 'alice', got %q", result.Spec.OwnerName)
	}
}

// --- TestWorkspaceStorage_Create_OwnerNotFound ---

func TestWorkspaceStorage_Create_OwnerNotFound(t *testing.T) {
	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			return nil, apierrors.NewNotFound("user", "999")
		},
	}

	wsStore := &mockWorkspaceStore{}

	storage := NewWorkspaceStorage(wsStore, userStore, nil)

	inputWs := &Workspace{
		Spec: WorkspaceSpec{
			DisplayName: "Test Workspace",
			OwnerID:     "999",
			Status:      "active",
		},
	}
	inputWs.ObjectMeta.Name = "test-workspace"

	_, err := storage.Create(context.Background(), inputWs, &rest.CreateOptions{})
	if err == nil {
		t.Fatal("expected error when owner not found, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestWorkspaceStorage_Update ---

func TestWorkspaceStorage_Update(t *testing.T) {
	updatedDBWs := testWorkspace(1, "updated-workspace", 10)
	updatedDBWs.DisplayName = "Updated Workspace"
	updatedDBWs.Description = "Updated description"

	wsStore := &mockWorkspaceStore{
		UpdateFn: func(ctx context.Context, ws *DBWorkspace) (*DBWorkspace, error) {
			if ws.ID != 1 {
				t.Errorf("expected workspace ID 1, got %d", ws.ID)
			}
			if ws.DisplayName != "Updated Workspace" {
				t.Errorf("expected displayName 'Updated Workspace', got %q", ws.DisplayName)
			}
			if ws.OwnerID != 10 {
				t.Errorf("expected ownerID 10, got %d", ws.OwnerID)
			}
			return updatedDBWs, nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, nil, nil)

	inputWs := &Workspace{
		Spec: WorkspaceSpec{
			DisplayName: "Updated Workspace",
			Description: "Updated description",
			OwnerID:     "10",
			Status:      "active",
		},
	}
	inputWs.ObjectMeta.Name = "updated-workspace"

	obj, err := storage.Update(context.Background(), inputWs, &rest.UpdateOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := obj.(*Workspace)
	if !ok {
		t.Fatalf("expected *Workspace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.DisplayName != "Updated Workspace" {
		t.Errorf("expected DisplayName 'Updated Workspace', got %q", result.Spec.DisplayName)
	}
	if result.Spec.Description != "Updated description" {
		t.Errorf("expected Description 'Updated description', got %q", result.Spec.Description)
	}
	if result.TypeMeta.Kind != "Workspace" {
		t.Errorf("expected Kind 'Workspace', got %q", result.TypeMeta.Kind)
	}
}

// --- TestWorkspaceStorage_Patch ---

func TestWorkspaceStorage_Patch(t *testing.T) {
	patchedDBWs := testWorkspace(1, "my-workspace", 10)
	patchedDBWs.DisplayName = "Patched Workspace"
	patchedDBWs.Description = "Original description"

	var patchCalled bool

	wsStore := &mockWorkspaceStore{
		PatchFn: func(ctx context.Context, id int64, ws *DBWorkspace) (*DBWorkspace, error) {
			patchCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			if ws.DisplayName != "Patched Workspace" {
				t.Errorf("expected displayName 'Patched Workspace', got %q", ws.DisplayName)
			}
			// Description should be empty (zero value) since patch input did not set it
			if ws.Description != "" {
				t.Errorf("expected description '' (zero value), got %q", ws.Description)
			}
			return patchedDBWs, nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, nil, nil)

	// Patch only the displayName
	inputWs := &Workspace{
		Spec: WorkspaceSpec{
			DisplayName: "Patched Workspace",
		},
	}

	obj, err := storage.Patch(context.Background(), inputWs, &rest.PatchOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !patchCalled {
		t.Error("expected wsStore.Patch to be called")
	}

	result, ok := obj.(*Workspace)
	if !ok {
		t.Fatalf("expected *Workspace, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.DisplayName != "Patched Workspace" {
		t.Errorf("expected DisplayName 'Patched Workspace', got %q", result.Spec.DisplayName)
	}
}

// --- TestWorkspaceStorage_Delete ---

func TestWorkspaceStorage_Delete(t *testing.T) {
	deleteCalled := false

	wsStore := &mockWorkspaceStore{
		DeleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			return nil
		},
	}

	storage := NewWorkspaceStorage(wsStore, nil, nil)

	err := storage.Delete(context.Background(), &rest.DeleteOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("expected Delete to be called on the store")
	}
}

// --- TestWorkspaceStorage_DeleteCollection ---

func TestWorkspaceStorage_DeleteCollection(t *testing.T) {
	wsStore := &mockWorkspaceStore{
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

	storage := NewWorkspaceStorage(wsStore, nil, nil)

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
