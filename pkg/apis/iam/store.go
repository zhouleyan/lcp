package iam

import (
	"context"

	"lcp.io/lcp/pkg/db"
)

// UserStore defines database operations on users.
type UserStore interface {
	Create(ctx context.Context, user *DBUser) (*DBUser, error)
	GetByID(ctx context.Context, id int64) (*DBUser, error)
	GetByUsername(ctx context.Context, username string) (*DBUser, error)
	GetByEmail(ctx context.Context, email string) (*DBUser, error)
	GetByPhone(ctx context.Context, phone string) (*DBUser, error)
	Update(ctx context.Context, user *DBUser) (*DBUser, error)
	Patch(ctx context.Context, id int64, user *DBUser) (*DBUser, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBUserWithNamespaces], error)
	GetUserForAuth(ctx context.Context, identifier string) (*DBUserForAuth, error)
	SetPasswordHash(ctx context.Context, id int64, hash string) error
}

// RefreshTokenStore defines database operations on refresh tokens.
type RefreshTokenStore interface {
	Create(ctx context.Context, token *DBRefreshToken) (*DBRefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*DBRefreshToken, error)
	ConsumeByHash(ctx context.Context, tokenHash string) (*DBRefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeByUserID(ctx context.Context, userID int64) error
	DeleteExpired(ctx context.Context) error
}

// WorkspaceStore defines database operations on workspaces.
type WorkspaceStore interface {
	Create(ctx context.Context, ws *DBWorkspace) (*DBWorkspaceWithOwner, error)
	GetByID(ctx context.Context, id int64) (*DBWorkspaceWithOwner, error)
	GetByName(ctx context.Context, name string) (*DBWorkspace, error)
	Update(ctx context.Context, ws *DBWorkspace) (*DBWorkspace, error)
	Patch(ctx context.Context, id int64, ws *DBWorkspace) (*DBWorkspace, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwner], error)
	CountNamespaces(ctx context.Context, workspaceID int64) (int64, error)
}

// NamespaceStore defines database operations on namespaces.
type NamespaceStore interface {
	Create(ctx context.Context, ns *DBNamespace) (*DBNamespaceWithOwner, error)
	GetByID(ctx context.Context, id int64) (*DBNamespaceWithOwner, error)
	GetByName(ctx context.Context, name string) (*DBNamespace, error)
	Update(ctx context.Context, ns *DBNamespace) (*DBNamespace, error)
	Patch(ctx context.Context, id int64, ns *DBNamespace) (*DBNamespace, error)
	Delete(ctx context.Context, id int64) error
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwner], error)
	CountUsers(ctx context.Context, namespaceID int64) (int64, error)
}

// PermissionCodeScope holds a permission code and its scope.
type PermissionCodeScope struct {
	Code  string
	Scope string
}

// PermissionStore defines database operations on permissions.
type PermissionStore interface {
	Upsert(ctx context.Context, perm *DBPermission) (*DBPermission, error)
	DeleteByModuleNotInCodeScopes(ctx context.Context, modulePrefix string, keepCodeScopes []string) error
	GetByCode(ctx context.Context, code, scope string) (*DBPermission, error)
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBPermission], error)
	ListAllCodes(ctx context.Context) ([]string, error)
	ListCodeScopes(ctx context.Context) ([]PermissionCodeScope, error)
	// SyncModule batch-upserts all permissions for a module and removes stale ones in a single transaction.
	SyncModule(ctx context.Context, modulePrefix string, perms []DBPermission) error
}

// RoleStore defines database operations on roles.
type RoleStore interface {
	Create(ctx context.Context, role *DBRole) (*DBRole, error)
	GetByID(ctx context.Context, id int64) (*DBRoleWithRules, error)
	GetByName(ctx context.Context, name string) (*DBRole, error)
	GetByNameAndWorkspace(ctx context.Context, name string, workspaceID int64) (*DBRole, error)
	GetByNameAndNamespace(ctx context.Context, name string, namespaceID int64) (*DBRole, error)
	Update(ctx context.Context, role *DBRole) (*DBRole, error)
	Upsert(ctx context.Context, role *DBRole) (*DBRole, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBRoleListRow], error)
	SetPermissionRules(ctx context.Context, roleID int64, patterns []string) error
	// SeedRBAC upserts built-in roles with rules and creates initial role bindings in a single transaction.
	SeedRBAC(ctx context.Context, roles []BuiltinRoleDef, adminUsername string) error
}

// RoleBindingStore defines database operations on role bindings.
type RoleBindingStore interface {
	Create(ctx context.Context, rb *DBRoleBinding) (*DBRoleBinding, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*DBRoleBinding, error)
	ListPlatform(ctx context.Context, query db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error)
	ListByWorkspaceID(ctx context.Context, workspaceID int64, query db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error)
	ListByNamespaceID(ctx context.Context, namespaceID int64, query db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error)
	ListByUserID(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBRoleBindingWithDetails], error)
	CountByRoleAndScope(ctx context.Context, roleID int64, scope string) (int64, error)
	GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error)
	GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error)
	GetUserIDsByWorkspaceID(ctx context.Context, workspaceID int64) ([]int64, error)
	GetUserIDsByNamespaceID(ctx context.Context, namespaceID int64) ([]int64, error)
	LoadUserPermissionRules(ctx context.Context, userID int64) ([]UserPermissionRuleRow, error)
	GetUserRoleBindingsWithRules(ctx context.Context, userID int64) ([]UserRoleBindingWithRules, error)
	// TransferOwnership transfers ownership of a workspace or namespace to a new user.
	// callerID + callerIsPlatformAdmin are used for authorization within the transaction.
	// Returns the old owner's user ID. The new owner must already be a member.
	TransferOwnership(ctx context.Context, scope string, resourceID int64, callerID int64, callerIsPlatformAdmin bool, newOwnerUserID int64, adminRoleName string) (oldOwnerUserID int64, err error)
	// Member management (replacing legacy join tables)
	// roleID=0 means use the default role (workspace-viewer / namespace-viewer).
	AddWorkspaceMember(ctx context.Context, userID, workspaceID int64, roleID int64) error
	AddNamespaceMember(ctx context.Context, userID, namespaceID int64, roleID int64) error
	RemoveWorkspaceMember(ctx context.Context, userID, workspaceID int64) error
	RemoveNamespaceMember(ctx context.Context, userID, namespaceID int64) error
	ListWorkspaceMembers(ctx context.Context, workspaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error)
	ListNamespaceMembers(ctx context.Context, namespaceID int64, query db.ListQuery) (*db.ListResult[DBUserWithRole], error)
	ListUserWorkspaces(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBWorkspaceWithOwnerAndRole], error)
	ListUserNamespaces(ctx context.Context, userID int64, query db.ListQuery) (*db.ListResult[DBNamespaceWithOwnerAndRole], error)
}
