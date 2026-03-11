package iam

import (
	"context"
	"testing"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// ===== workspaceUserStorage tests =====

// --- TestWorkspaceUserStorage_List ---

func TestWorkspaceUserStorage_List(t *testing.T) {
	rbStore := &mockRoleBindingStore{
		ListWorkspaceMembersFn: func(ctx context.Context, workspaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error) {
			if workspaceID != 1 {
				t.Errorf("expected workspaceID 1, got %d", workspaceID)
			}
			return &db.ListResult[DBUserWithRole]{
				Items: []DBUserWithRole{
					{
						User: generated.User{
							ID:          10,
							Username:    "alice",
							Email:       "alice@example.com",
							DisplayName: "Alice",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						Role:     RoleWorkspaceAdmin,
						JoinedAt: testTime,
					},
					{
						User: generated.User{
							ID:          20,
							Username:    "bob",
							Email:       "bob@example.com",
							DisplayName: "Bob",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						Role:     RoleWorkspaceViewer,
						JoinedAt: testTime,
					},
				},
				TotalCount: 2,
			}, nil
		},
	}

	s := NewWorkspaceUserStorage(rbStore, nil)
	lister := s.(rest.Lister)

	obj, err := lister.List(context.Background(), &rest.ListOptions{
		PathParams: map[string]string{"workspaceId": "1"},
		Filters:    map[string]string{},
		Pagination: rest.Pagination{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userList, ok := obj.(*UserList)
	if !ok {
		t.Fatalf("expected *UserList, got %T", obj)
	}

	if userList.TypeMeta.Kind != "UserList" {
		t.Errorf("expected Kind 'UserList', got %q", userList.TypeMeta.Kind)
	}
	if userList.TotalCount != 2 {
		t.Errorf("expected TotalCount 2, got %d", userList.TotalCount)
	}
	if len(userList.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(userList.Items))
	}

	// Verify first user
	if userList.Items[0].ObjectMeta.ID != "10" {
		t.Errorf("expected first item ID '10', got %q", userList.Items[0].ObjectMeta.ID)
	}
	if userList.Items[0].ObjectMeta.Name != "alice" {
		t.Errorf("expected first item Name 'alice', got %q", userList.Items[0].ObjectMeta.Name)
	}
	if userList.Items[0].Spec.Email != "alice@example.com" {
		t.Errorf("expected first item Email 'alice@example.com', got %q", userList.Items[0].Spec.Email)
	}

	// Verify second user
	if userList.Items[1].ObjectMeta.ID != "20" {
		t.Errorf("expected second item ID '20', got %q", userList.Items[1].ObjectMeta.ID)
	}
	if userList.Items[1].ObjectMeta.Name != "bob" {
		t.Errorf("expected second item Name 'bob', got %q", userList.Items[1].ObjectMeta.Name)
	}
}

// --- TestWorkspaceUserStorage_List_InvalidID ---

func TestWorkspaceUserStorage_List_InvalidID(t *testing.T) {
	s := NewWorkspaceUserStorage(&mockRoleBindingStore{}, nil)
	lister := s.(rest.Lister)

	_, err := lister.List(context.Background(), &rest.ListOptions{
		PathParams: map[string]string{"workspaceId": "abc"},
		Filters:    map[string]string{},
		Pagination: rest.Pagination{Page: 1, PageSize: 20},
	})
	if err == nil {
		t.Fatal("expected error for invalid workspace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestWorkspaceUserStorage_Create ---

func TestWorkspaceUserStorage_Create(t *testing.T) {
	var addCalls []int64
	var getUserCalls []int64

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			getUserCalls = append(getUserCalls, id)
			return testUser(id, "user", "user@example.com"), nil
		},
	}

	rbStore := &mockRoleBindingStore{
		AddWorkspaceMemberFn: func(ctx context.Context, userID, workspaceID int64, roleID int64) error {
			addCalls = append(addCalls, userID)
			if workspaceID != 1 {
				t.Errorf("expected workspaceID 1, got %d", workspaceID)
			}
			return nil
		},
	}

	s := NewWorkspaceUserStorage(rbStore, userStore)
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"10", "20", "30"}}

	obj, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify each user was verified
	if len(getUserCalls) != 3 {
		t.Errorf("expected 3 userStore.GetByID calls, got %d", len(getUserCalls))
	}

	// Verify each user was added
	if len(addCalls) != 3 {
		t.Errorf("expected 3 rbStore.AddWorkspaceMember calls, got %d", len(addCalls))
	}

	result, ok := obj.(*rest.DeletionResult)
	if !ok {
		t.Fatalf("expected *rest.DeletionResult, got %T", obj)
	}
	if result.SuccessCount != 3 {
		t.Errorf("expected SuccessCount 3, got %d", result.SuccessCount)
	}
}

// --- TestWorkspaceUserStorage_Create_UserNotFound ---

func TestWorkspaceUserStorage_Create_UserNotFound(t *testing.T) {
	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			return nil, apierrors.NewNotFound("user", "999")
		},
	}

	rbStore := &mockRoleBindingStore{}

	s := NewWorkspaceUserStorage(rbStore, userStore)
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"999"}}

	_, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err == nil {
		t.Fatal("expected error when user not found, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestWorkspaceUserStorage_Create_InvalidWorkspaceID ---

func TestWorkspaceUserStorage_Create_InvalidWorkspaceID(t *testing.T) {
	s := NewWorkspaceUserStorage(&mockRoleBindingStore{}, &mockUserStore{})
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"10"}}

	_, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"workspaceId": "abc"},
	})
	if err == nil {
		t.Fatal("expected error for invalid workspace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestWorkspaceUserStorage_DeleteCollection ---

func TestWorkspaceUserStorage_DeleteCollection(t *testing.T) {
	var removeCalls []int64

	rbStore := &mockRoleBindingStore{
		RemoveWorkspaceMemberFn: func(ctx context.Context, userID, workspaceID int64) error {
			removeCalls = append(removeCalls, userID)
			if workspaceID != 1 {
				t.Errorf("expected workspaceID 1, got %d", workspaceID)
			}
			return nil
		},
	}

	s := NewWorkspaceUserStorage(rbStore, nil)
	deleter := s.(rest.CollectionDeleter)

	result, err := deleter.DeleteCollection(context.Background(), []string{"10", "20"}, &rest.DeleteOptions{
		PathParams: map[string]string{"workspaceId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(removeCalls) != 2 {
		t.Errorf("expected 2 rbStore.RemoveWorkspaceMember calls, got %d", len(removeCalls))
	}
	if removeCalls[0] != 10 {
		t.Errorf("expected first Remove call with userID 10, got %d", removeCalls[0])
	}
	if removeCalls[1] != 20 {
		t.Errorf("expected second Remove call with userID 20, got %d", removeCalls[1])
	}

	if result.SuccessCount != 2 {
		t.Errorf("expected SuccessCount 2, got %d", result.SuccessCount)
	}
}

// --- TestWorkspaceUserStorage_DeleteCollection_InvalidWorkspaceID ---

func TestWorkspaceUserStorage_DeleteCollection_InvalidWorkspaceID(t *testing.T) {
	s := NewWorkspaceUserStorage(&mockRoleBindingStore{}, nil)
	deleter := s.(rest.CollectionDeleter)

	_, err := deleter.DeleteCollection(context.Background(), []string{"10"}, &rest.DeleteOptions{
		PathParams: map[string]string{"workspaceId": "abc"},
	})
	if err == nil {
		t.Fatal("expected error for invalid workspace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// ===== namespaceUserStorage tests =====

// --- TestNamespaceUserStorage_List ---

func TestNamespaceUserStorage_List(t *testing.T) {
	rbStore := &mockRoleBindingStore{
		ListNamespaceMembersFn: func(ctx context.Context, namespaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error) {
			if namespaceID != 5 {
				t.Errorf("expected namespaceID 5, got %d", namespaceID)
			}
			return &db.ListResult[DBUserWithRole]{
				Items: []DBUserWithRole{
					{
						User: generated.User{
							ID:          10,
							Username:    "alice",
							Email:       "alice@example.com",
							DisplayName: "Alice",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						Role:     RoleNamespaceAdmin,
						JoinedAt: testTime,
					},
					{
						User: generated.User{
							ID:          20,
							Username:    "bob",
							Email:       "bob@example.com",
							DisplayName: "Bob",
							Status:      "active",
							CreatedAt:   testTime,
							UpdatedAt:   testTime,
						},
						Role:     RoleNamespaceViewer,
						JoinedAt: testTime,
					},
				},
				TotalCount: 2,
			}, nil
		},
	}

	s := NewNamespaceUserStorage(rbStore, nil, nil)
	lister := s.(rest.Lister)

	obj, err := lister.List(context.Background(), &rest.ListOptions{
		PathParams: map[string]string{"namespaceId": "5"},
		Filters:    map[string]string{},
		Pagination: rest.Pagination{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userList, ok := obj.(*UserList)
	if !ok {
		t.Fatalf("expected *UserList, got %T", obj)
	}

	if userList.TypeMeta.Kind != "UserList" {
		t.Errorf("expected Kind 'UserList', got %q", userList.TypeMeta.Kind)
	}
	if userList.TotalCount != 2 {
		t.Errorf("expected TotalCount 2, got %d", userList.TotalCount)
	}
	if len(userList.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(userList.Items))
	}

	// Verify first user
	if userList.Items[0].ObjectMeta.ID != "10" {
		t.Errorf("expected first item ID '10', got %q", userList.Items[0].ObjectMeta.ID)
	}
	if userList.Items[0].ObjectMeta.Name != "alice" {
		t.Errorf("expected first item Name 'alice', got %q", userList.Items[0].ObjectMeta.Name)
	}
	if userList.Items[0].Spec.Email != "alice@example.com" {
		t.Errorf("expected first item Email 'alice@example.com', got %q", userList.Items[0].Spec.Email)
	}

	// Verify second user
	if userList.Items[1].ObjectMeta.ID != "20" {
		t.Errorf("expected second item ID '20', got %q", userList.Items[1].ObjectMeta.ID)
	}
	if userList.Items[1].Spec.Username != "bob" {
		t.Errorf("expected second item Username 'bob', got %q", userList.Items[1].Spec.Username)
	}
}

// --- TestNamespaceUserStorage_List_InvalidID ---

func TestNamespaceUserStorage_List_InvalidID(t *testing.T) {
	s := NewNamespaceUserStorage(&mockRoleBindingStore{}, nil, nil)
	lister := s.(rest.Lister)

	_, err := lister.List(context.Background(), &rest.ListOptions{
		PathParams: map[string]string{"namespaceId": "abc"},
		Filters:    map[string]string{},
		Pagination: rest.Pagination{Page: 1, PageSize: 20},
	})
	if err == nil {
		t.Fatal("expected error for invalid namespace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceUserStorage_Create ---

func TestNamespaceUserStorage_Create(t *testing.T) {
	var addCalls []int64
	var getUserCalls []int64

	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			if id != 5 {
				t.Errorf("expected namespace ID 5, got %d", id)
			}
			// MaxMembers=0 means unlimited
			return testNamespaceWithOwner(5, "my-ns", 1, 10, "alice", "my-ws"), nil
		},
		CountUsersFn: func(ctx context.Context, namespaceID int64) (int64, error) {
			t.Error("CountUsers should not be called when MaxMembers is 0")
			return 0, nil
		},
	}

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			getUserCalls = append(getUserCalls, id)
			return testUser(id, "user", "user@example.com"), nil
		},
	}

	rbStore := &mockRoleBindingStore{
		AddNamespaceMemberFn: func(ctx context.Context, userID, namespaceID int64, roleID int64) error {
			addCalls = append(addCalls, userID)
			if namespaceID != 5 {
				t.Errorf("expected namespaceID 5, got %d", namespaceID)
			}
			return nil
		},
	}

	s := NewNamespaceUserStorage(rbStore, nsStore, userStore)
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"10", "20"}}

	obj, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"namespaceId": "5"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify each user was verified
	if len(getUserCalls) != 2 {
		t.Errorf("expected 2 userStore.GetByID calls, got %d", len(getUserCalls))
	}

	// Verify each user was added
	if len(addCalls) != 2 {
		t.Errorf("expected 2 rbStore.AddNamespaceMember calls, got %d", len(addCalls))
	}

	result, ok := obj.(*rest.DeletionResult)
	if !ok {
		t.Fatalf("expected *rest.DeletionResult, got %T", obj)
	}
	if result.SuccessCount != 2 {
		t.Errorf("expected SuccessCount 2, got %d", result.SuccessCount)
	}
}

// --- TestNamespaceUserStorage_Create_ExceedsMaxUsers ---

func TestNamespaceUserStorage_Create_ExceedsMaxUsers(t *testing.T) {
	nsWithOwner := testNamespaceWithOwner(5, "my-ns", 1, 10, "alice", "my-ws")
	nsWithOwner.MaxMembers = 5 // max 5 members

	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			return nsWithOwner, nil
		},
		CountUsersFn: func(ctx context.Context, namespaceID int64) (int64, error) {
			return 3, nil // currently 3 members
		},
	}

	s := NewNamespaceUserStorage(&mockRoleBindingStore{}, nsStore, &mockUserStore{})
	creator := s.(rest.Creator)

	// Try to add 3 users when there are already 3 and max is 5 (3+3=6 > 5)
	req := &BatchRequest{IDs: []string{"20", "30", "40"}}

	_, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"namespaceId": "5"},
	})
	if err == nil {
		t.Fatal("expected error when exceeding max members, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceUserStorage_Create_WithinMaxUsers ---

func TestNamespaceUserStorage_Create_WithinMaxUsers(t *testing.T) {
	nsWithOwner := testNamespaceWithOwner(5, "my-ns", 1, 10, "alice", "my-ws")
	nsWithOwner.MaxMembers = 10 // max 10 members

	var countUsersCalled bool

	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			return nsWithOwner, nil
		},
		CountUsersFn: func(ctx context.Context, namespaceID int64) (int64, error) {
			countUsersCalled = true
			return 3, nil // currently 3 members, adding 2 = 5 <= 10
		},
	}

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			return testUser(id, "user", "user@example.com"), nil
		},
	}

	rbStore := &mockRoleBindingStore{
		AddNamespaceMemberFn: func(ctx context.Context, userID, namespaceID int64, roleID int64) error {
			return nil
		},
	}

	s := NewNamespaceUserStorage(rbStore, nsStore, userStore)
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"20", "30"}}

	obj, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"namespaceId": "5"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !countUsersCalled {
		t.Error("expected nsStore.CountUsers to be called when MaxMembers > 0")
	}

	result, ok := obj.(*rest.DeletionResult)
	if !ok {
		t.Fatalf("expected *rest.DeletionResult, got %T", obj)
	}
	if result.SuccessCount != 2 {
		t.Errorf("expected SuccessCount 2, got %d", result.SuccessCount)
	}
}

// --- TestNamespaceUserStorage_Create_UserNotFound ---

func TestNamespaceUserStorage_Create_UserNotFound(t *testing.T) {
	nsStore := &mockNamespaceStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
			return testNamespaceWithOwner(5, "my-ns", 1, 10, "alice", "my-ws"), nil
		},
	}

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			return nil, apierrors.NewNotFound("user", "999")
		},
	}

	s := NewNamespaceUserStorage(&mockRoleBindingStore{}, nsStore, userStore)
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"999"}}

	_, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"namespaceId": "5"},
	})
	if err == nil {
		t.Fatal("expected error when user not found, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceUserStorage_Create_InvalidNamespaceID ---

func TestNamespaceUserStorage_Create_InvalidNamespaceID(t *testing.T) {
	s := NewNamespaceUserStorage(&mockRoleBindingStore{}, &mockNamespaceStore{}, &mockUserStore{})
	creator := s.(rest.Creator)

	req := &BatchRequest{IDs: []string{"10"}}

	_, err := creator.Create(context.Background(), req, &rest.CreateOptions{
		PathParams: map[string]string{"namespaceId": "abc"},
	})
	if err == nil {
		t.Fatal("expected error for invalid namespace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

// --- TestNamespaceUserStorage_DeleteCollection ---

func TestNamespaceUserStorage_DeleteCollection(t *testing.T) {
	var removeCalls []int64

	rbStore := &mockRoleBindingStore{
		RemoveNamespaceMemberFn: func(ctx context.Context, userID, namespaceID int64) error {
			removeCalls = append(removeCalls, userID)
			if namespaceID != 5 {
				t.Errorf("expected namespaceID 5, got %d", namespaceID)
			}
			return nil
		},
	}

	s := NewNamespaceUserStorage(rbStore, nil, nil)
	deleter := s.(rest.CollectionDeleter)

	result, err := deleter.DeleteCollection(context.Background(), []string{"10", "20", "30"}, &rest.DeleteOptions{
		PathParams: map[string]string{"namespaceId": "5"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(removeCalls) != 3 {
		t.Errorf("expected 3 rbStore.RemoveNamespaceMember calls, got %d", len(removeCalls))
	}
	expectedIDs := []int64{10, 20, 30}
	for i, id := range removeCalls {
		if id != expectedIDs[i] {
			t.Errorf("expected Remove call %d with userID %d, got %d", i, expectedIDs[i], id)
		}
	}

	if result.SuccessCount != 3 {
		t.Errorf("expected SuccessCount 3, got %d", result.SuccessCount)
	}
}

// --- TestNamespaceUserStorage_DeleteCollection_InvalidNamespaceID ---

func TestNamespaceUserStorage_DeleteCollection_InvalidNamespaceID(t *testing.T) {
	s := NewNamespaceUserStorage(&mockRoleBindingStore{}, nil, nil)
	deleter := s.(rest.CollectionDeleter)

	_, err := deleter.DeleteCollection(context.Background(), []string{"10"}, &rest.DeleteOptions{
		PathParams: map[string]string{"namespaceId": "abc"},
	})
	if err == nil {
		t.Fatal("expected error for invalid namespace ID, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}
