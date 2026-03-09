package iam

import (
	"context"
	"time"

	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

// --- Mock UserStore ---

type mockUserStore struct {
	CreateFn          func(ctx context.Context, user *DBUser) (*DBUser, error)
	GetByIDFn         func(ctx context.Context, id int64) (*DBUser, error)
	GetByUsernameFn   func(ctx context.Context, username string) (*DBUser, error)
	GetByEmailFn      func(ctx context.Context, email string) (*DBUser, error)
	GetByPhoneFn      func(ctx context.Context, phone string) (*DBUser, error)
	UpdateFn          func(ctx context.Context, user *DBUser) (*DBUser, error)
	PatchFn           func(ctx context.Context, id int64, user *DBUser) (*DBUser, error)
	UpdateLastLoginFn func(ctx context.Context, id int64) error
	DeleteFn          func(ctx context.Context, id int64) error
	DeleteByIDsFn     func(ctx context.Context, ids []int64) (int64, error)
	ListFn            func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBUserWithNamespaces], error)
	GetUserForAuthFn  func(ctx context.Context, identifier string) (*DBUserForAuth, error)
	SetPasswordHashFn func(ctx context.Context, id int64, hash string) error
}

func (m *mockUserStore) Create(ctx context.Context, user *DBUser) (*DBUser, error) {
	return m.CreateFn(ctx, user)
}
func (m *mockUserStore) GetByID(ctx context.Context, id int64) (*DBUser, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockUserStore) GetByUsername(ctx context.Context, username string) (*DBUser, error) {
	return m.GetByUsernameFn(ctx, username)
}
func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*DBUser, error) {
	return m.GetByEmailFn(ctx, email)
}
func (m *mockUserStore) GetByPhone(ctx context.Context, phone string) (*DBUser, error) {
	return m.GetByPhoneFn(ctx, phone)
}
func (m *mockUserStore) Update(ctx context.Context, user *DBUser) (*DBUser, error) {
	return m.UpdateFn(ctx, user)
}
func (m *mockUserStore) Patch(ctx context.Context, id int64, user *DBUser) (*DBUser, error) {
	return m.PatchFn(ctx, id, user)
}
func (m *mockUserStore) UpdateLastLogin(ctx context.Context, id int64) error {
	return m.UpdateLastLoginFn(ctx, id)
}
func (m *mockUserStore) Delete(ctx context.Context, id int64) error {
	return m.DeleteFn(ctx, id)
}
func (m *mockUserStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return m.DeleteByIDsFn(ctx, ids)
}
func (m *mockUserStore) List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBUserWithNamespaces], error) {
	return m.ListFn(ctx, query)
}
func (m *mockUserStore) GetUserForAuth(ctx context.Context, identifier string) (*DBUserForAuth, error) {
	return m.GetUserForAuthFn(ctx, identifier)
}
func (m *mockUserStore) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	return m.SetPasswordHashFn(ctx, id, hash)
}

// --- Mock RefreshTokenStore ---

type mockRefreshTokenStore struct {
	CreateFn        func(ctx context.Context, token *DBRefreshToken) (*DBRefreshToken, error)
	GetByHashFn     func(ctx context.Context, tokenHash string) (*DBRefreshToken, error)
	ConsumeByHashFn func(ctx context.Context, tokenHash string) (*DBRefreshToken, error)
	RevokeFn        func(ctx context.Context, tokenHash string) error
	RevokeByUserIDFn func(ctx context.Context, userID int64) error
	DeleteExpiredFn func(ctx context.Context) error
}

func (m *mockRefreshTokenStore) Create(ctx context.Context, token *DBRefreshToken) (*DBRefreshToken, error) {
	return m.CreateFn(ctx, token)
}
func (m *mockRefreshTokenStore) GetByHash(ctx context.Context, tokenHash string) (*DBRefreshToken, error) {
	return m.GetByHashFn(ctx, tokenHash)
}
func (m *mockRefreshTokenStore) ConsumeByHash(ctx context.Context, tokenHash string) (*DBRefreshToken, error) {
	return m.ConsumeByHashFn(ctx, tokenHash)
}
func (m *mockRefreshTokenStore) Revoke(ctx context.Context, tokenHash string) error {
	return m.RevokeFn(ctx, tokenHash)
}
func (m *mockRefreshTokenStore) RevokeByUserID(ctx context.Context, userID int64) error {
	return m.RevokeByUserIDFn(ctx, userID)
}
func (m *mockRefreshTokenStore) DeleteExpired(ctx context.Context) error {
	return m.DeleteExpiredFn(ctx)
}

// --- Mock WorkspaceStore ---

type mockWorkspaceStore struct {
	CreateFn          func(ctx context.Context, ws *DBWorkspace) (*DBWorkspaceWithOwner, error)
	GetByIDFn         func(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error)
	GetByNameFn       func(ctx context.Context, name string) (*DBWorkspace, error)
	UpdateFn          func(ctx context.Context, ws *DBWorkspace) (*DBWorkspace, error)
	PatchFn           func(ctx context.Context, id int64, ws *DBWorkspace) (*DBWorkspace, error)
	DeleteFn          func(ctx context.Context, id int64) error
	DeleteByIDsFn     func(ctx context.Context, ids []int64) (int64, error)
	ListFn            func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwner], error)
	CountNamespacesFn func(ctx context.Context, workspaceID int64) (int64, error)
}

func (m *mockWorkspaceStore) Create(ctx context.Context, ws *DBWorkspace) (*DBWorkspaceWithOwner, error) {
	return m.CreateFn(ctx, ws)
}
func (m *mockWorkspaceStore) GetByID(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockWorkspaceStore) GetByName(ctx context.Context, name string) (*DBWorkspace, error) {
	return m.GetByNameFn(ctx, name)
}
func (m *mockWorkspaceStore) Update(ctx context.Context, ws *DBWorkspace) (*DBWorkspace, error) {
	return m.UpdateFn(ctx, ws)
}
func (m *mockWorkspaceStore) Patch(ctx context.Context, id int64, ws *DBWorkspace) (*DBWorkspace, error) {
	return m.PatchFn(ctx, id, ws)
}
func (m *mockWorkspaceStore) Delete(ctx context.Context, id int64) error {
	return m.DeleteFn(ctx, id)
}
func (m *mockWorkspaceStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return m.DeleteByIDsFn(ctx, ids)
}
func (m *mockWorkspaceStore) List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwner], error) {
	return m.ListFn(ctx, query)
}
func (m *mockWorkspaceStore) CountNamespaces(ctx context.Context, workspaceID int64) (int64, error) {
	return m.CountNamespacesFn(ctx, workspaceID)
}

// --- Mock NamespaceStore ---

type mockNamespaceStore struct {
	CreateFn      func(ctx context.Context, ns *DBNamespace) (*DBNamespaceWithOwner, error)
	GetByIDFn     func(ctx context.Context, id int64) (*DBNamespaceWithOwner, error)
	GetByNameFn   func(ctx context.Context, name string) (*DBNamespace, error)
	UpdateFn      func(ctx context.Context, ns *DBNamespace) (*DBNamespace, error)
	PatchFn       func(ctx context.Context, id int64, ns *DBNamespace) (*DBNamespace, error)
	DeleteFn      func(ctx context.Context, id int64) error
	DeleteByIDsFn func(ctx context.Context, ids []int64) (int64, error)
	ListFn        func(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error)
	CountUsersFn  func(ctx context.Context, namespaceID int64) (int64, error)
}

func (m *mockNamespaceStore) Create(ctx context.Context, ns *DBNamespace) (*DBNamespaceWithOwner, error) {
	return m.CreateFn(ctx, ns)
}
func (m *mockNamespaceStore) GetByID(ctx context.Context, id int64) (*DBNamespaceWithOwner, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockNamespaceStore) GetByName(ctx context.Context, name string) (*DBNamespace, error) {
	return m.GetByNameFn(ctx, name)
}
func (m *mockNamespaceStore) Update(ctx context.Context, ns *DBNamespace) (*DBNamespace, error) {
	return m.UpdateFn(ctx, ns)
}
func (m *mockNamespaceStore) Patch(ctx context.Context, id int64, ns *DBNamespace) (*DBNamespace, error) {
	return m.PatchFn(ctx, id, ns)
}
func (m *mockNamespaceStore) Delete(ctx context.Context, id int64) error {
	return m.DeleteFn(ctx, id)
}
func (m *mockNamespaceStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return m.DeleteByIDsFn(ctx, ids)
}
func (m *mockNamespaceStore) List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error) {
	return m.ListFn(ctx, query)
}
func (m *mockNamespaceStore) CountUsers(ctx context.Context, namespaceID int64) (int64, error) {
	return m.CountUsersFn(ctx, namespaceID)
}

// mockRoleBindingStore provides a mock for RoleBindingStore.
type mockRoleBindingStore struct {
	LoadUserPermissionRulesFn      func(ctx context.Context, userID int64) ([]UserPermissionRuleRow, error)
	GetAccessibleWorkspaceIDsFn    func(ctx context.Context, userID int64) ([]int64, error)
	GetAccessibleNamespaceIDsFn    func(ctx context.Context, userID int64) ([]int64, error)
	GetUserRoleBindingsWithRulesFn func(ctx context.Context, userID int64) ([]UserRoleBindingWithRules, error)
	AddWorkspaceMemberFn           func(ctx context.Context, userID, workspaceID int64) error
	AddNamespaceMemberFn           func(ctx context.Context, userID, namespaceID int64) error
	RemoveWorkspaceMemberFn        func(ctx context.Context, userID, workspaceID int64) error
	RemoveNamespaceMemberFn        func(ctx context.Context, userID, namespaceID int64) error
	ListWorkspaceMembersFn         func(ctx context.Context, workspaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error)
	ListNamespaceMembersFn         func(ctx context.Context, namespaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error)
	ListUserWorkspacesFn           func(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwnerAndRole], error)
	ListUserNamespacesFn           func(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwnerAndRole], error)
}

func (m *mockRoleBindingStore) LoadUserPermissionRules(ctx context.Context, userID int64) ([]UserPermissionRuleRow, error) {
	return m.LoadUserPermissionRulesFn(ctx, userID)
}
func (m *mockRoleBindingStore) GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	return m.GetAccessibleWorkspaceIDsFn(ctx, userID)
}
func (m *mockRoleBindingStore) GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	return m.GetAccessibleNamespaceIDsFn(ctx, userID)
}
func (m *mockRoleBindingStore) GetUserRoleBindingsWithRules(ctx context.Context, userID int64) ([]UserRoleBindingWithRules, error) {
	return m.GetUserRoleBindingsWithRulesFn(ctx, userID)
}
func (m *mockRoleBindingStore) Create(context.Context, *DBRoleBinding) (*DBRoleBinding, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) Delete(context.Context, int64) error { panic("not implemented") }
func (m *mockRoleBindingStore) GetByID(context.Context, int64) (*DBRoleBinding, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListPlatform(context.Context, db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListByWorkspaceID(context.Context, int64, db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListByNamespaceID(context.Context, int64, db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListByUserID(context.Context, int64, db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) CountByRoleAndScope(context.Context, int64, string) (int64, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) GetUserIDsByWorkspaceID(context.Context, int64) ([]int64, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) GetUserIDsByNamespaceID(context.Context, int64) ([]int64, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) TransferOwnership(context.Context, string, int64, int64, bool, int64, string) (int64, error) {
	panic("not implemented")
}
func (m *mockRoleBindingStore) AddWorkspaceMember(ctx context.Context, userID, workspaceID int64) error {
	if m.AddWorkspaceMemberFn != nil {
		return m.AddWorkspaceMemberFn(ctx, userID, workspaceID)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) AddNamespaceMember(ctx context.Context, userID, namespaceID int64) error {
	if m.AddNamespaceMemberFn != nil {
		return m.AddNamespaceMemberFn(ctx, userID, namespaceID)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) RemoveWorkspaceMember(ctx context.Context, userID, workspaceID int64) error {
	if m.RemoveWorkspaceMemberFn != nil {
		return m.RemoveWorkspaceMemberFn(ctx, userID, workspaceID)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) RemoveNamespaceMember(ctx context.Context, userID, namespaceID int64) error {
	if m.RemoveNamespaceMemberFn != nil {
		return m.RemoveNamespaceMemberFn(ctx, userID, namespaceID)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListWorkspaceMembers(ctx context.Context, workspaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error) {
	if m.ListWorkspaceMembersFn != nil {
		return m.ListWorkspaceMembersFn(ctx, workspaceID, query)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListNamespaceMembers(ctx context.Context, namespaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error) {
	if m.ListNamespaceMembersFn != nil {
		return m.ListNamespaceMembersFn(ctx, namespaceID, query)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListUserWorkspaces(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwnerAndRole], error) {
	if m.ListUserWorkspacesFn != nil {
		return m.ListUserWorkspacesFn(ctx, userID, query)
	}
	panic("not implemented")
}
func (m *mockRoleBindingStore) ListUserNamespaces(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwnerAndRole], error) {
	if m.ListUserNamespacesFn != nil {
		return m.ListUserNamespacesFn(ctx, userID, query)
	}
	panic("not implemented")
}

// --- Test data helpers ---

var testTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// testUser creates a DBUser with sensible defaults for testing.
func testUser(id int64, username, email string) *DBUser {
	return &DBUser{
		ID:           id,
		Username:     username,
		Email:        email,
		DisplayName:  username,
		Phone:        "",
		AvatarUrl:    "",
		Status:       "active",
		PasswordHash: "",
		LastLoginAt:  nil,
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
}

// testWorkspace creates a DBWorkspace with sensible defaults for testing.
func testWorkspace(id int64, name string, ownerID int64) *DBWorkspace {
	return &DBWorkspace{
		ID:          id,
		Name:        name,
		DisplayName: name,
		Description: "",
		OwnerID:     ownerID,
		Status:      "active",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}
}

// testNamespace creates a DBNamespace with sensible defaults for testing.
func testNamespace(id int64, name string, workspaceID, ownerID int64) *DBNamespace {
	return &DBNamespace{
		ID:          id,
		Name:        name,
		DisplayName: name,
		Description: "",
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
		Visibility:  "private",
		MaxMembers:  0,
		Status:      "active",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}
}

// testWorkspaceWithOwner creates a DBWorkspaceWithOwner for testing.
func testWorkspaceWithOwner(id int64, name string, ownerID int64, ownerUsername string) *DBWorkspaceWithOwner {
	return &DBWorkspaceWithOwner{
		Workspace: generated.Workspace{
			ID:          id,
			Name:        name,
			DisplayName: name,
			Description: "",
			OwnerID:     ownerID,
			Status:      "active",
			CreatedAt:   testTime,
			UpdatedAt:   testTime,
		},
		OwnerUsername:   ownerUsername,
		NamespaceCount: 0,
		MemberCount:    0,
	}
}

// testNamespaceWithOwner creates a DBNamespaceWithOwner for testing.
func testNamespaceWithOwner(id int64, name string, workspaceID, ownerID int64, ownerUsername, workspaceName string) *DBNamespaceWithOwner {
	return &DBNamespaceWithOwner{
		Namespace: generated.Namespace{
			ID:          id,
			Name:        name,
			DisplayName: name,
			Description: "",
			WorkspaceID: workspaceID,
			OwnerID:     ownerID,
			Visibility:  "private",
			MaxMembers:  0,
			Status:      "active",
			CreatedAt:   testTime,
			UpdatedAt:   testTime,
		},
		OwnerUsername:  ownerUsername,
		WorkspaceName: workspaceName,
		MemberCount:   0,
	}
}
