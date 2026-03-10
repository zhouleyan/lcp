export interface TypeMeta {
  apiVersion: string
  kind: string
}

export interface ObjectMeta {
  id: string
  name: string
  createdAt: string
  updatedAt: string
}

// --- User ---

export interface UserSpec {
  username: string
  email: string
  displayName?: string
  phone?: string
  avatarUrl?: string
  status?: "active" | "inactive"
  namespaces?: string[]
  role?: string
  joinedAt?: string
}

export interface User extends TypeMeta {
  metadata: ObjectMeta
  spec: UserSpec
}

export interface UserList extends TypeMeta {
  items: User[]
  totalCount: number
}

// --- Workspace ---

export interface WorkspaceSpec {
  displayName?: string
  description?: string
  ownerId: string
  ownerName?: string
  namespaceCount?: number
  memberCount?: number
  status?: "active" | "inactive"
  role?: string
  roleDisplayName?: string
  joinedAt?: string
}

export interface Workspace extends TypeMeta {
  metadata: ObjectMeta
  spec: WorkspaceSpec
}

export interface WorkspaceList extends TypeMeta {
  items: Workspace[]
  totalCount: number
}

// --- Namespace ---

export interface NamespaceSpec {
  displayName?: string
  description?: string
  workspaceId: string
  workspaceName?: string
  ownerId: string
  ownerName?: string
  visibility?: "public" | "private"
  maxMembers?: number
  memberCount?: number
  status?: "active" | "inactive"
  role?: string
  roleDisplayName?: string
  joinedAt?: string
}

export interface Namespace extends TypeMeta {
  metadata: ObjectMeta
  spec: NamespaceSpec
}

export interface NamespaceList extends TypeMeta {
  items: Namespace[]
  totalCount: number
}

// --- Common ---

export interface BatchRequest extends TypeMeta {
  ids: string[]
}

export interface ListParams {
  page?: number
  pageSize?: number
  sortBy?: string
  sortOrder?: "asc" | "desc"
  [key: string]: string | number | undefined
}

export interface ChangePasswordRequest {
  oldPassword: string
  newPassword: string
}

// --- Permission ---

export interface PermissionSpec {
  code: string
  method: string
  path: string
  scope: "platform" | "workspace" | "namespace"
  description?: string
}

export interface Permission extends TypeMeta {
  metadata: ObjectMeta
  spec: PermissionSpec
}

export interface PermissionList extends TypeMeta {
  items: Permission[]
  totalCount: number
}

// --- Role ---

export interface RoleSpec {
  name: string
  displayName?: string
  description?: string
  scope: "platform" | "workspace" | "namespace"
  builtin?: boolean
  ruleCount?: number
  rules?: string[]
}

export interface Role extends TypeMeta {
  metadata: ObjectMeta
  spec: RoleSpec
}

export interface RoleList extends TypeMeta {
  items: Role[]
  totalCount: number
}

// --- RoleBinding ---

export interface RoleBindingSpec {
  userId: string
  roleId: string
  scope: "platform" | "workspace" | "namespace"
  workspaceId?: string
  namespaceId?: string
  isOwner?: boolean
  roleName?: string
  roleDisplayName?: string
  username?: string
  userDisplayName?: string
}

export interface RoleBinding extends TypeMeta {
  metadata: ObjectMeta
  spec: RoleBindingSpec
}

export interface RoleBindingList extends TypeMeta {
  items: RoleBinding[]
  totalCount: number
}

// --- UserPermissions ---

export interface WorkspaceScopePerms {
  roleNames: string[]
  permissions: string[]
}

export interface NamespaceScopePerms {
  roleNames: string[]
  workspaceId: string
  permissions: string[]
}

export interface UserPermissionsSpec {
  isPlatformAdmin: boolean
  platform: string[]
  workspaces: Record<string, WorkspaceScopePerms>
  namespaces: Record<string, NamespaceScopePerms>
}

export interface UserPermissions extends TypeMeta {
  spec: UserPermissionsSpec
}

// --- Overview ---

export interface OverviewSpec {
  workspaceCount: number
  namespaceCount: number
  userCount: number
  memberCount: number
  roleCount: number
}

export interface Overview extends TypeMeta {
  spec: OverviewSpec
}

// --- Transfer Ownership ---

export interface TransferOwnershipRequest {
  newOwnerUserId: string
}

export interface OIDCUserInfo {
  sub: string
  name?: string
  email?: string
  phone_number?: string
}

export interface StatusResponseDetail {
  field: string
  message: string
}

export interface StatusResponse extends TypeMeta {
  status: string | number
  reason: string
  message: string
  details?: StatusResponseDetail[]
}
