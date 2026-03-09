package store

import (
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
)

// NewStores creates all IAM store instances from a database connection.
// Adding a new store only requires adding a line here and a field in iam.Stores.
func NewStores(database *db.DB) iam.Stores {
	return iam.Stores{
		User:          NewPGUserStore(database.Queries),
		Workspace:     NewPGWorkspaceStore(database.Pool, database.Queries),
		Namespace:     NewPGNamespaceStore(database.Pool, database.Queries),
		UserWorkspace: NewPGUserWorkspaceStore(database.Queries),
		UserNamespace: NewPGUserNamespaceStore(database.Pool, database.Queries),
		RefreshToken:  NewPGRefreshTokenStore(database.Queries),
		Permission:    NewPGPermissionStore(database.Pool, database.Queries),
		Role:          NewPGRoleStore(database.Pool, database.Queries),
		RoleBinding:   NewPGRoleBindingStore(database.Pool, database.Queries),
	}
}
