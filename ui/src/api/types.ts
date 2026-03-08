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
