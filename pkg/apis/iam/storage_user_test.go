package iam

import (
	"context"
	"testing"
	"time"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// --- TestUserStorage_Get ---

func TestUserStorage_Get(t *testing.T) {
	dbUser := testUser(1, "alice", "alice@example.com")

	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			if id != 1 {
				t.Fatalf("expected id 1, got %d", id)
			}
			return dbUser, nil
		},
	}

	uwStore := &mockUserWorkspaceStore{
		ListByUserIDFn: func(ctx context.Context, userID int64) ([]DBWorkspaceWithRole, error) {
			return []DBWorkspaceWithRole{
				{
					Workspace: generated.Workspace{
						ID:          10,
						Name:        "ws-one",
						DisplayName: "Workspace One",
					},
					Role:     "owner",
					JoinedAt: testTime,
				},
			}, nil
		},
	}

	unStore := &mockUserNamespaceStore{
		ListByUserIDFn: func(ctx context.Context, userID int64) ([]DBNamespaceWithRole, error) {
			return []DBNamespaceWithRole{
				{
					Namespace: generated.Namespace{
						ID:          20,
						Name:        "ns-one",
						DisplayName: "Namespace One",
						WorkspaceID: 10,
					},
					Role:     "member",
					JoinedAt: testTime,
				},
			}, nil
		},
	}

	storage := NewUserStorage(userStore, uwStore, unStore)

	obj, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"userId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user, ok := obj.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", obj)
	}

	// Verify basic fields
	if user.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", user.ObjectMeta.ID)
	}
	if user.ObjectMeta.Name != "alice" {
		t.Errorf("expected Name 'alice', got %q", user.ObjectMeta.Name)
	}
	if user.Spec.Username != "alice" {
		t.Errorf("expected Username 'alice', got %q", user.Spec.Username)
	}
	if user.Spec.Email != "alice@example.com" {
		t.Errorf("expected Email 'alice@example.com', got %q", user.Spec.Email)
	}
	if user.Spec.Status != "active" {
		t.Errorf("expected Status 'active', got %q", user.Spec.Status)
	}
	if user.TypeMeta.Kind != "User" {
		t.Errorf("expected Kind 'User', got %q", user.TypeMeta.Kind)
	}

	// Verify enriched workspaces
	if len(user.Spec.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(user.Spec.Workspaces))
	}
	ws := user.Spec.Workspaces[0]
	if ws.ID != "10" {
		t.Errorf("expected workspace ID '10', got %q", ws.ID)
	}
	if ws.Name != "ws-one" {
		t.Errorf("expected workspace Name 'ws-one', got %q", ws.Name)
	}
	if ws.DisplayName != "Workspace One" {
		t.Errorf("expected workspace DisplayName 'Workspace One', got %q", ws.DisplayName)
	}
	if ws.Role != "owner" {
		t.Errorf("expected workspace Role 'owner', got %q", ws.Role)
	}
	expectedJoinedAt := testTime.Format(time.RFC3339)
	if ws.JoinedAt != expectedJoinedAt {
		t.Errorf("expected workspace JoinedAt %q, got %q", expectedJoinedAt, ws.JoinedAt)
	}

	// Verify enriched namespaces
	if len(user.Spec.NamespaceRefs) != 1 {
		t.Fatalf("expected 1 namespace ref, got %d", len(user.Spec.NamespaceRefs))
	}
	ns := user.Spec.NamespaceRefs[0]
	if ns.ID != "20" {
		t.Errorf("expected namespace ID '20', got %q", ns.ID)
	}
	if ns.Name != "ns-one" {
		t.Errorf("expected namespace Name 'ns-one', got %q", ns.Name)
	}
	if ns.DisplayName != "Namespace One" {
		t.Errorf("expected namespace DisplayName 'Namespace One', got %q", ns.DisplayName)
	}
	if ns.WorkspaceID != "10" {
		t.Errorf("expected namespace WorkspaceID '10', got %q", ns.WorkspaceID)
	}
	if ns.Role != "member" {
		t.Errorf("expected namespace Role 'member', got %q", ns.Role)
	}
	if ns.JoinedAt != expectedJoinedAt {
		t.Errorf("expected namespace JoinedAt %q, got %q", expectedJoinedAt, ns.JoinedAt)
	}
}

func TestUserStorage_Get_InvalidID(t *testing.T) {
	storage := NewUserStorage(&mockUserStore{}, &mockUserWorkspaceStore{}, &mockUserNamespaceStore{})

	_, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"userId": "abc"},
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

func TestUserStorage_Get_NotFound(t *testing.T) {
	userStore := &mockUserStore{
		GetByIDFn: func(ctx context.Context, id int64) (*DBUser, error) {
			return nil, apierrors.NewNotFound("user", "999")
		},
	}

	storage := NewUserStorage(userStore, &mockUserWorkspaceStore{}, &mockUserNamespaceStore{})

	_, err := storage.Get(context.Background(), &rest.GetOptions{
		PathParams: map[string]string{"userId": "999"},
	})
	if err == nil {
		t.Fatal("expected error for not found user, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

// --- TestUserStorage_List ---

func TestUserStorage_List(t *testing.T) {
	userStore := &mockUserStore{
		ListFn: func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBUserWithNamespaces], error) {
			return &db.ListResult[DBUserWithNamespaces]{
				Items: []DBUserWithNamespaces{
					{
						User:           *testUser(1, "alice", "alice@example.com"),
						NamespaceNames: []string{"ns-one", "ns-two"},
					},
					{
						User:           *testUser(2, "bob", "bob@example.com"),
						NamespaceNames: nil,
					},
				},
				TotalCount: 2,
			}, nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

	obj, err := storage.List(context.Background(), &rest.ListOptions{
		Filters: map[string]string{"status": "active"},
		Pagination: rest.Pagination{
			Page:     1,
			PageSize: 20,
		},
		SortBy:    "username",
		SortOrder: rest.SortOrderAsc,
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

	// First user should have namespace names
	if len(userList.Items[0].Spec.Namespaces) != 2 {
		t.Errorf("expected 2 namespaces for first user, got %d", len(userList.Items[0].Spec.Namespaces))
	}
	if userList.Items[0].Spec.Username != "alice" {
		t.Errorf("expected first user 'alice', got %q", userList.Items[0].Spec.Username)
	}

	// Second user should have no namespaces
	if len(userList.Items[1].Spec.Namespaces) != 0 {
		t.Errorf("expected 0 namespaces for second user, got %d", len(userList.Items[1].Spec.Namespaces))
	}
}

// --- TestUserStorage_Create ---

func TestUserStorage_Create(t *testing.T) {
	createdUser := testUser(1, "alice", "alice@example.com")
	createdUser.Phone = "13800138000"

	userStore := &mockUserStore{
		CreateFn: func(ctx context.Context, user *DBUser) (*DBUser, error) {
			if user.Username != "alice" {
				t.Errorf("expected username 'alice', got %q", user.Username)
			}
			if user.Email != "alice@example.com" {
				t.Errorf("expected email 'alice@example.com', got %q", user.Email)
			}
			return createdUser, nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

	inputUser := &User{
		Spec: UserSpec{
			Username: "alice",
			Email:    "alice@example.com",
			Phone:    "13800138000",
			Status:   "active",
		},
	}

	obj, err := storage.Create(context.Background(), inputUser, &rest.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := obj.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.Username != "alice" {
		t.Errorf("expected Username 'alice', got %q", result.Spec.Username)
	}
	if result.TypeMeta.Kind != "User" {
		t.Errorf("expected Kind 'User', got %q", result.TypeMeta.Kind)
	}
}

func TestUserStorage_Create_ValidationFails(t *testing.T) {
	storage := NewUserStorage(&mockUserStore{}, nil, nil)

	// Missing required fields: username, email, phone
	inputUser := &User{
		Spec: UserSpec{},
	}

	_, err := storage.Create(context.Background(), inputUser, &rest.CreateOptions{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if statusErr.Status != 400 {
		t.Errorf("expected status 400, got %d", statusErr.Status)
	}
}

func TestUserStorage_Create_WithPassword(t *testing.T) {
	createdUser := testUser(1, "alice", "alice@example.com")
	createdUser.Phone = "13800138000"

	var capturedHash string
	var setPasswordCalled bool

	userStore := &mockUserStore{
		CreateFn: func(ctx context.Context, user *DBUser) (*DBUser, error) {
			return createdUser, nil
		},
		SetPasswordHashFn: func(ctx context.Context, id int64, hash string) error {
			setPasswordCalled = true
			capturedHash = hash
			if id != 1 {
				t.Errorf("expected user ID 1, got %d", id)
			}
			return nil
		},
	}

	hashCalled := false
	hasher := func(password string) (string, error) {
		hashCalled = true
		if password != "Test1234" {
			t.Errorf("expected password 'Test1234', got %q", password)
		}
		return "hashed-Test1234", nil
	}

	storage := NewUserStorageWithPassword(userStore, nil, nil, hasher)

	inputUser := &User{
		Spec: UserSpec{
			Username: "alice",
			Email:    "alice@example.com",
			Phone:    "13800138000",
			Password: "Test1234",
			Status:   "active",
		},
	}

	obj, err := storage.Create(context.Background(), inputUser, &rest.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hashCalled {
		t.Error("expected password hasher to be called")
	}
	if !setPasswordCalled {
		t.Error("expected SetPasswordHash to be called")
	}
	if capturedHash != "hashed-Test1234" {
		t.Errorf("expected hash 'hashed-Test1234', got %q", capturedHash)
	}

	result, ok := obj.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", obj)
	}
	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
}

// --- TestUserStorage_Update ---

func TestUserStorage_Update(t *testing.T) {
	updatedUser := testUser(1, "alice_new", "alice_new@example.com")
	updatedUser.Phone = "13800138001"

	userStore := &mockUserStore{
		UpdateFn: func(ctx context.Context, user *DBUser) (*DBUser, error) {
			if user.ID != 1 {
				t.Errorf("expected user ID 1, got %d", user.ID)
			}
			if user.Username != "alice_new" {
				t.Errorf("expected username 'alice_new', got %q", user.Username)
			}
			return updatedUser, nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

	inputUser := &User{
		Spec: UserSpec{
			Username: "alice_new",
			Email:    "alice_new@example.com",
			Phone:    "13800138001",
			Status:   "active",
		},
	}

	obj, err := storage.Update(context.Background(), inputUser, &rest.UpdateOptions{
		PathParams: map[string]string{"userId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := obj.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", obj)
	}

	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.Username != "alice_new" {
		t.Errorf("expected Username 'alice_new', got %q", result.Spec.Username)
	}
	if result.Spec.Email != "alice_new@example.com" {
		t.Errorf("expected Email 'alice_new@example.com', got %q", result.Spec.Email)
	}
}

// --- TestUserStorage_Patch ---

func TestUserStorage_Patch(t *testing.T) {
	patchedUser := testUser(1, "alice", "alice@example.com")
	patchedUser.DisplayName = "Alice Updated"

	userStore := &mockUserStore{
		PatchFn: func(ctx context.Context, id int64, user *DBUser) (*DBUser, error) {
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			if user.DisplayName != "Alice Updated" {
				t.Errorf("expected DisplayName 'Alice Updated', got %q", user.DisplayName)
			}
			return patchedUser, nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

	inputUser := &User{
		Spec: UserSpec{
			DisplayName: "Alice Updated",
		},
	}

	obj, err := storage.Patch(context.Background(), inputUser, &rest.PatchOptions{
		PathParams: map[string]string{"userId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := obj.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", obj)
	}

	if result.Spec.DisplayName != "Alice Updated" {
		t.Errorf("expected DisplayName 'Alice Updated', got %q", result.Spec.DisplayName)
	}
	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
}

// --- TestUserStorage_Delete ---

func TestUserStorage_Delete(t *testing.T) {
	deleteCalled := false

	userStore := &mockUserStore{
		DeleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			if id != 1 {
				t.Errorf("expected id 1, got %d", id)
			}
			return nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

	err := storage.Delete(context.Background(), &rest.DeleteOptions{
		PathParams: map[string]string{"userId": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("expected Delete to be called on the store")
	}
}

// --- TestUserStorage_DeleteCollection ---

func TestUserStorage_DeleteCollection(t *testing.T) {
	userStore := &mockUserStore{
		DeleteByIDsFn: func(ctx context.Context, ids []int64) (int64, error) {
			if len(ids) != 3 {
				t.Errorf("expected 3 IDs, got %d", len(ids))
			}
			// Simulate that 2 out of 3 were actually deleted
			return 2, nil
		},
	}

	storage := NewUserStorage(userStore, nil, nil)

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

func TestUserStorage_DeleteCollection_InvalidID(t *testing.T) {
	storage := NewUserStorage(&mockUserStore{}, nil, nil)

	_, err := storage.DeleteCollection(context.Background(), []string{"1", "abc", "3"}, &rest.DeleteOptions{})
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
