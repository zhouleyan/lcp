package iam

import (
	"time"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// --- User types ---

// User
// +openapi:description=User is the API representation of a user resource.
// +openapi:path=/users
// +openapi:path=/namespaces/{namespaceId}/users
type User struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             UserSpec `json:"spec"`
}

func (u *User) GetTypeMeta() *runtime.TypeMeta { return &u.TypeMeta }

// UserSpec
// +openapi:description=UserSpec holds user-specific fields.
type UserSpec struct {
	// +openapi:required
	// +openapi:description=Username must be 3-50 alphanumeric characters or underscores
	Username string `json:"username"`
	// +openapi:required
	// +openapi:description=Valid email address
	// +openapi:format=email
	Email string `json:"email"`
	// +openapi:description=Display name for the user
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=Phone number in E.164 format
	Phone string `json:"phone,omitempty"`
	// +openapi:description=URL to the user's avatar image
	// +openapi:format=uri
	AvatarURL string `json:"avatarUrl,omitempty"`
	// +openapi:description=Account status
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// UserList
// +openapi:description=UserList is a paginated list of users.
type UserList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []User `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

func (u *UserList) GetTypeMeta() *runtime.TypeMeta { return &u.TypeMeta }

// --- Namespace types ---

// Namespace
// +openapi:description=Namespace is the API representation of a namespace resource.
// +openapi:path=/namespaces
type Namespace struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             NamespaceSpec `json:"spec"`
}

func (n *Namespace) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NamespaceSpec
// +openapi:description=NamespaceSpec holds namespace-specific fields.
type NamespaceSpec struct {
	// +openapi:description=Display name for the namespace
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=Description of the namespace
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=ID of the namespace owner
	OwnerID string `json:"ownerId"`
	// +openapi:description=Namespace visibility
	// +openapi:enum=public,private
	Visibility string `json:"visibility,omitempty"`
	// +openapi:description=Maximum number of members allowed
	MaxMembers int `json:"maxMembers,omitempty"`
	// +openapi:description=Namespace status
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// NamespaceList
// +openapi:description=NamespaceList is a paginated list of namespaces.
type NamespaceList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Namespace `json:"items"`
	TotalCount       int64       `json:"totalCount"`
}

func (n *NamespaceList) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// --- DB type aliases ---

// DBUser is an alias for the sqlc-generated User model.
type DBUser = generated.User

// DBNamespace is an alias for the sqlc-generated Namespace model.
type DBNamespace = generated.Namespace

// DBUserNamespace is an alias for the sqlc-generated UserNamespace model.
type DBUserNamespace = generated.UserNamespace

// DBUserWithNamespaces extends the generated User with associated namespace names.
type DBUserWithNamespaces struct {
	generated.User
	NamespaceNames []string `json:"namespace_names"`
}

// DBNamespaceWithOwner extends Namespace with owner username.
type DBNamespaceWithOwner struct {
	generated.Namespace
	OwnerUsername string `json:"owner_username"`
}

// DBNamespaceWithRole is a namespace with the user's role in it.
type DBNamespaceWithRole struct {
	generated.Namespace
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// DBUserWithRole is a user with their role in a namespace.
type DBUserWithRole struct {
	generated.User
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
