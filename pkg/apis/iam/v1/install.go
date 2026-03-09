package v1

import (
	"context"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of IAM module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewIAMModule initializes the IAM module: builds the API group,
// syncs permissions to DB, and seeds built-in roles.
func NewIAMModule(ctx context.Context, database *db.DB) ModuleResult {
	group, p := newAPIGroupInfo(database)

	// Sync permissions to DB
	if _, err := iam.SyncPermissions(ctx, p.Permission, []*rest.APIGroupInfo{group}); err != nil {
		logger.Fatalf("cannot sync IAM permissions: %v", err)
	}

	// Seed built-in roles, permission rules, and initial bindings
	if err := iam.SeedRBAC(ctx, p.Role); err != nil {
		logger.Fatalf("cannot seed RBAC: %v", err)
	}

	return ModuleResult{
		Group: group,
	}
}

// newAPIGroupInfo initializes the full IAM storage stack and builds the API group.
func newAPIGroupInfo(database *db.DB) (*rest.APIGroupInfo, *iam.RESTStorageProvider) {
	p := iam.NewRESTStorageProvider(iamstore.NewStores(database))

	userStorage, userActions := newUserStorage(p)
	checker := iam.NewRBACChecker(p.RoleBinding)

	wsStorage := iam.NewWorkspaceStorage(p.Workspace, p.User, p.RoleBinding, checker)
	nsStorage := iam.NewNamespaceStorage(p.Namespace, p.Workspace, p.User, p.RoleBinding, checker)
	wsUserStorage := iam.NewWorkspaceUserStorage(p.RoleBinding, p.User)
	nsUserStorage := iam.NewNamespaceUserStorage(p.RoleBinding, p.Namespace, p.User)
	permStorage := iam.NewPermissionStorage(p.Permission)
	roleStorage := iam.NewRoleStorage(p.Role, p.RoleBinding)
	rbStorage := iam.NewRoleBindingStorage(p.RoleBinding, p.Role)
	wsRbStorage := iam.NewWorkspaceRoleBindingStorage(p.RoleBinding, p.Role)
	nsRbStorage := iam.NewNamespaceRoleBindingStorage(p.RoleBinding, p.Role, p.Namespace)
	wsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, iam.ScopeWorkspace)
	nsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, iam.ScopeNamespace)

	group := &rest.APIGroupInfo{
		GroupName: "iam",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "users",
				Storage: userStorage,
				Actions: userActions,
				CustomVerbs: []rest.CustomVerbInfo{
					{Name: "workspaces", Storage: iam.NewUserWorkspacesVerb(p.RoleBinding)},
					{Name: "namespaces", Storage: iam.NewUserNamespacesVerb(p.RoleBinding)},
					{Name: "rolebindings", Storage: iam.NewUserRoleBindingsVerb(p.RoleBinding)},
					{Name: "permissions", Storage: iam.NewUserPermissionsVerb(p.RoleBinding, p.Permission)},
				},
			},
			{
				Name:    "workspaces",
				Storage: wsStorage,
				Actions: []rest.ActionInfo{
					{
						Name:    "transfer-ownership",
						Method:  "POST",
						Handler: iam.NewTransferOwnershipHandler(p.RoleBinding, checker),
					},
				},
				SubResources: []rest.ResourceInfo{
					{
						Name:    "namespaces",
						Storage: nsStorage,
						Actions: []rest.ActionInfo{
							{
								Name:    "transfer-ownership",
								Method:  "POST",
								Handler: iam.NewNamespaceTransferOwnershipHandler(p.RoleBinding, checker),
							},
						},
						SubResources: []rest.ResourceInfo{
							{Name: "users", Storage: nsUserStorage},
							{Name: "rolebindings", Storage: nsRbStorage},
							{Name: "roles", Storage: nsRoleStorage},
						},
					},
					{Name: "users", Storage: wsUserStorage},
					{Name: "rolebindings", Storage: wsRbStorage},
					{Name: "roles", Storage: wsRoleStorage},
				},
			},
			{
				Name:    "namespaces",
				Storage: nsStorage,
				Actions: []rest.ActionInfo{
					{
						Name:    "transfer-ownership",
						Method:  "POST",
						Handler: iam.NewNamespaceTransferOwnershipHandler(p.RoleBinding, checker),
					},
				},
				SubResources: []rest.ResourceInfo{
					{Name: "users", Storage: nsUserStorage},
					{Name: "rolebindings", Storage: nsRbStorage},
					{Name: "roles", Storage: nsRoleStorage},
				},
			},
			{
				Name:    "permissions",
				Storage: permStorage,
			},
			{
				Name:    "roles",
				Storage: roleStorage,
			},
			{
				Name:    "rolebindings",
				Storage: rbStorage,
			},
		},
	}

	return group, p
}

// newUserStorage creates the user REST storage with password hashing and change-password action.
func newUserStorage(p *iam.RESTStorageProvider) (rest.StandardStorage, []rest.ActionInfo) {
	ps := iam.NewPasswordService()
	storage := iam.NewUserStorageWithPassword(p.User, ps.Hash)
	actions := []rest.ActionInfo{
		{
			Name:    "change-password",
			Method:  "POST",
			Handler: iam.NewChangePasswordHandler(p.User, p.RefreshToken, ps.Hash, ps.Verify),
		},
	}
	return storage, actions
}
