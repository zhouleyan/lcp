package iam

// Stores holds all IAM Store instances.
// Adding a new store only requires adding a field here.
type Stores struct {
	User          UserStore
	Workspace     WorkspaceStore
	Namespace     NamespaceStore
	UserWorkspace UserWorkspaceStore
	UserNamespace UserNamespaceStore
	RefreshToken  RefreshTokenStore
	Permission    PermissionStore
	Role          RoleStore
	RoleBinding   RoleBindingStore
}

// RESTStorageProvider centralizes all IAM Store instances.
// Downstream code (e.g. v1/install.go) accesses stores via exported fields.
type RESTStorageProvider struct {
	Stores
}

// NewRESTStorageProvider creates a RESTStorageProvider from a Stores struct.
func NewRESTStorageProvider(s Stores) *RESTStorageProvider {
	return &RESTStorageProvider{Stores: s}
}
