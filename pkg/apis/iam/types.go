package iam

import (
	"time"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// --- User types ---

// User
// +openapi:description=用户管理：平台用户的增删改查，支持密码设置与修改。用户可被添加为工作空间或项目的成员。
type User struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             UserSpec `json:"spec"`
}

func (u *User) GetTypeMeta() *runtime.TypeMeta { return &u.TypeMeta }

// UserSpec
// +openapi:description=用户属性：包含用户名、邮箱、显示名称、手机号、头像、密码等字段。
type UserSpec struct {
	// +openapi:required
	// +openapi:description=用户名，3-50个字符，仅支持小写字母、数字和下划线
	Username string `json:"username"`
	// +openapi:required
	// +openapi:description=邮箱地址
	// +openapi:format=email
	Email string `json:"email"`
	// +openapi:description=用户显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=手机号码
	Phone string `json:"phone,omitempty"`
	// +openapi:description=用户头像 URL
	// +openapi:format=uri
	AvatarURL string `json:"avatarUrl,omitempty"`
	// +openapi:description=用户密码（只写字段，创建时设置，响应中不返回）
	// +openapi:format=password
	Password string `json:"password,omitempty"`
	// +openapi:description=账户状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=用户所属的项目列表（仅在列表查询时返回）
	Namespaces []string `json:"namespaces,omitempty"`
}

// UserList
// +openapi:description=用户列表：分页返回的用户集合。
type UserList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []User `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

func (u *UserList) GetTypeMeta() *runtime.TypeMeta { return &u.TypeMeta }

// --- Workspace types ---

// Workspace
// +openapi:description=工作空间管理：工作空间是平台的顶层租户/组织单元，包含多个项目（Namespace）和成员。创建工作空间时会自动创建默认项目并将创建者设为所有者。
type Workspace struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             WorkspaceSpec `json:"spec"`
}

func (w *Workspace) GetTypeMeta() *runtime.TypeMeta { return &w.TypeMeta }

// WorkspaceSpec
// +openapi:description=工作空间属性：包含显示名称、描述、所有者和状态。
type WorkspaceSpec struct {
	// +openapi:description=工作空间显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=工作空间描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=工作空间所有者的用户 ID
	OwnerID string `json:"ownerId"`
	// +openapi:description=工作空间状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// WorkspaceList
// +openapi:description=工作空间列表：分页返回的工作空间集合。
type WorkspaceList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Workspace `json:"items"`
	TotalCount       int64       `json:"totalCount"`
}

func (w *WorkspaceList) GetTypeMeta() *runtime.TypeMeta { return &w.TypeMeta }

// --- Namespace types ---

// Namespace
// +openapi:description=项目管理：项目（Namespace）是工作空间下的子单元，用于组织团队和资源。项目归属于某个工作空间，拥有独立的成员列表。添加项目成员时会自动将其加入父工作空间。
type Namespace struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             NamespaceSpec `json:"spec"`
}

func (n *Namespace) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NamespaceSpec
// +openapi:description=项目属性：包含显示名称、描述、所属工作空间、所有者、可见性、成员上限和状态。
type NamespaceSpec struct {
	// +openapi:description=项目显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=项目描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=所属工作空间 ID
	WorkspaceID string `json:"workspaceId"`
	// +openapi:required
	// +openapi:description=项目所有者的用户 ID
	OwnerID string `json:"ownerId"`
	// +openapi:description=项目可见性
	// +openapi:enum=public,private
	Visibility string `json:"visibility,omitempty"`
	// +openapi:description=项目最大成员数（0 表示不限制）
	MaxMembers int `json:"maxMembers,omitempty"`
	// +openapi:description=项目状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// NamespaceList
// +openapi:description=项目列表：分页返回的项目集合。
type NamespaceList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Namespace `json:"items"`
	TotalCount       int64       `json:"totalCount"`
}

func (n *NamespaceList) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// --- Batch request type ---

// BatchRequest 批量操作请求：用于批量添加或移除成员。
type BatchRequest struct {
	runtime.TypeMeta `json:",inline"`
	IDs              []string `json:"ids"`
}

func (b *BatchRequest) GetTypeMeta() *runtime.TypeMeta { return &b.TypeMeta }

// --- DB type aliases ---

// DBUser is an alias for the sqlc-generated User model.
type DBUser = generated.User

// DBWorkspace is an alias for the sqlc-generated Workspace model.
type DBWorkspace = generated.Workspace

// DBNamespace is an alias for the sqlc-generated Namespace model.
type DBNamespace = generated.Namespace

// DBUserWorkspace is an alias for the sqlc-generated UserWorkspace model.
type DBUserWorkspace = generated.UserWorkspace

// DBUserNamespace is an alias for the sqlc-generated UserNamespace model.
type DBUserNamespace = generated.UserNamespace

// DBUserWithNamespaces extends the generated User with associated namespace names.
type DBUserWithNamespaces struct {
	generated.User
	NamespaceNames []string `json:"namespace_names"`
}

// DBWorkspaceWithOwner extends Workspace with owner username.
type DBWorkspaceWithOwner struct {
	generated.Workspace
	OwnerUsername string `json:"owner_username"`
}

// DBWorkspaceWithRole is a workspace with the user's role in it.
type DBWorkspaceWithRole struct {
	generated.Workspace
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
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

// DBRefreshToken is an alias for the sqlc-generated RefreshToken model.
type DBRefreshToken = generated.RefreshToken

// DBUserForAuth is an alias for the sqlc-generated GetUserForAuthRow.
type DBUserForAuth = generated.GetUserForAuthRow
