package v1

import (
	"context"

	"lcp.io/lcp/lib/config"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	"lcp.io/lcp/pkg/apis/iam"
	iamstore "lcp.io/lcp/pkg/apis/iam/store"
	"lcp.io/lcp/pkg/db"
)

// ModuleResult holds the output of IAM module initialization.
type ModuleResult struct {
	Group *rest.APIGroupInfo
}

// NewIAMModule initializes the IAM module: builds the API group and seeds built-in roles.
// Permission sync is handled centrally by apis.NewAPIGroupInfos after all modules are registered.
func NewIAMModule(ctx context.Context, database *db.DB) ModuleResult {
	group, p := newAPIGroupInfo(database)

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
	roleStorage := iam.NewRoleStorage(p.Role, p.RoleBinding, p.Permission)
	rbStorage := iam.NewRoleBindingStorage(p.RoleBinding, p.Role, checker)
	wsRbStorage := iam.NewWorkspaceRoleBindingStorage(p.RoleBinding, p.Role, checker)
	nsRbStorage := iam.NewNamespaceRoleBindingStorage(p.RoleBinding, p.Role, p.Namespace, checker)
	wsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, p.Permission, iam.ScopeWorkspace)
	nsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, p.Permission, iam.ScopeNamespace)

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

// SyncAllPermissions syncs permissions for all API groups to the database.
func SyncAllPermissions(ctx context.Context, database *db.DB, groups []*rest.APIGroupInfo) {
	permStore := iamstore.NewPGPermissionStore(database.Pool, database.Queries)
	if _, err := iam.SyncPermissions(ctx, permStore, groups); err != nil {
		logger.Fatalf("cannot sync permissions: %v", err)
	}
}

// NewAuthorizer creates a fully-wired Authorizer from API group definitions and database.
func NewAuthorizer(database *db.DB, groups []*rest.APIGroupInfo) *filters.Authorizer {
	rbStore := iamstore.NewPGRoleBindingStore(database.Pool, database.Queries)
	nsStore := iamstore.NewPGNamespaceStore(database.Pool, database.Queries)
	return iam.NewAuthorizer(rbStore, nsStore, groups)
}

// NewOIDCProvider creates the OIDC provider with all internal store wiring.
// Returns nil if OIDC is not configured (no key files).
func NewOIDCProvider(database *db.DB, cfg *config.OIDCConfig) *oidc.Provider {
	if cfg.PrivateKeyFile == "" || cfg.PublicKeyFile == "" {
		logger.Infof("OIDC not configured (no key files), authentication disabled")
		return nil
	}

	providerCfg, err := oidc.ParseConfig(cfg)
	if err != nil {
		logger.Fatalf("invalid OIDC config: %v", err)
	}

	keySet, err := oidc.LoadKeySet(cfg.PrivateKeyFile, cfg.PublicKeyFile)
	if err != nil {
		logger.Fatalf("cannot load OIDC keys: %v", err)
	}

	userStore := iamstore.NewPGUserStore(database.Queries)
	refreshStore := iamstore.NewPGRefreshTokenStore(database.Queries)

	provider := oidc.NewProvider(providerCfg, keySet,
		iam.NewUserLookupAdapter(userStore),
		iam.NewRefreshTokenAdapter(refreshStore),
	)
	provider.SetClients(oidc.ParseClients(cfg.Clients))

	logger.Infof("OIDC provider initialized (issuer=%s)", cfg.Issuer)
	return provider
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
