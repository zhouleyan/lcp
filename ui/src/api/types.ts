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
  roleBindingCount?: number
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
  roleBindingCount?: number
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
  workspaceName?: string
  namespaceName?: string
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

// --- AuditLog ---

export interface AuditLogSpec {
  id: string
  userId?: string
  username: string
  eventType: "api_operation" | "authentication"
  action: string
  resourceType?: string
  resourceId?: string
  module?: string
  scope: "platform" | "workspace" | "namespace"
  workspaceId?: string
  namespaceId?: string
  httpMethod?: string
  httpPath?: string
  statusCode?: number
  clientIp?: string
  userAgent?: string
  durationMs?: number
  success: boolean
  detail?: Record<string, unknown>
  responseDetail?: Record<string, unknown>
  createdAt: string
}

export interface AuditLog extends TypeMeta {
  spec: AuditLogSpec
}

export interface AuditLogList extends TypeMeta {
  items: AuditLog[]
  totalCount: number
}

// --- Host ---

export interface HostSpec {
  displayName?: string
  description?: string
  hostname?: string
  ipAddress?: string
  os?: string
  arch?: string
  cpuCores?: number
  memoryMb?: number
  diskGb?: number
  labels?: Record<string, string>
  scope: "platform" | "workspace" | "namespace"
  workspaceId?: string
  namespaceId?: string
  environmentId?: string
  environmentName?: string
  workspaceName?: string
  namespaceName?: string
  status?: string
}

export interface Host extends TypeMeta {
  metadata: ObjectMeta
  spec: HostSpec
}

export interface HostList extends TypeMeta {
  items: Host[]
  totalCount: number
}

// --- Environment ---

export interface EnvironmentSpec {
  displayName?: string
  description?: string
  envType?: string
  scope: "platform" | "workspace" | "namespace"
  workspaceId?: string
  namespaceId?: string
  hostCount?: number
  status?: string
}

export interface Environment extends TypeMeta {
  metadata: ObjectMeta
  spec: EnvironmentSpec
}

export interface EnvironmentList extends TypeMeta {
  items: Environment[]
  totalCount: number
}

// --- Network ---

export interface NetworkSpec {
  displayName?: string
  description?: string
  cidr?: string
  maxSubnets?: number
  isPublic?: boolean
  status?: "active" | "inactive"
  subnetCount?: number
}

export interface Network extends TypeMeta {
  metadata: ObjectMeta
  spec: NetworkSpec
}

export interface NetworkList extends TypeMeta {
  items: Network[]
  totalCount: number
}

// --- Subnet ---

export interface SubnetSpec {
  displayName?: string
  description?: string
  cidr: string
  gateway?: string
  networkId?: string
  freeIPs?: number
  usedIPs?: number
  totalIPs?: number
  nextFreeIP?: string
}

export interface Subnet extends TypeMeta {
  metadata: ObjectMeta
  spec: SubnetSpec
}

export interface SubnetList extends TypeMeta {
  items: Subnet[]
  totalCount: number
}

// --- IPAllocation ---

export interface IPAllocationSpec {
  ip: string
  description?: string
  isGateway?: boolean
  subnetId?: string
}

export interface IPAllocation extends TypeMeta {
  metadata: ObjectMeta
  spec: IPAllocationSpec
}

export interface IPAllocationList extends TypeMeta {
  items: IPAllocation[]
  totalCount: number
}

// --- Infra Requests ---

export interface BindEnvironmentRequest {
  environmentId: string
}

// --- Region ---

export interface RegionSpec {
  displayName?: string
  description?: string
  status?: string
  latitude?: number | null
  longitude?: number | null
  siteCount?: number
}

export interface Region extends TypeMeta {
  metadata: ObjectMeta
  spec: RegionSpec
}

export interface RegionList extends TypeMeta {
  items: Region[]
  totalCount: number
}

// --- Site ---

export interface SiteSpec {
  displayName?: string
  description?: string
  regionId: string
  regionName?: string
  status?: string
  address?: string
  latitude?: number | null
  longitude?: number | null
  contactName?: string
  contactPhone?: string
  contactEmail?: string
  locationCount?: number
}

export interface Site extends TypeMeta {
  metadata: ObjectMeta
  spec: SiteSpec
}

export interface SiteList extends TypeMeta {
  items: Site[]
  totalCount: number
}

// --- Location ---

export interface LocationSpec {
  displayName?: string
  description?: string
  siteId: string
  siteName?: string
  regionId?: string
  regionName?: string
  status?: string
  floor?: string
  rackCapacity?: number
  rackCount?: number
  contactName?: string
  contactPhone?: string
  contactEmail?: string
}

export interface Location extends TypeMeta {
  metadata: ObjectMeta
  spec: LocationSpec
}

export interface LocationList extends TypeMeta {
  items: Location[]
  totalCount: number
}

// --- Rack ---

export interface RackSpec {
  displayName?: string
  description?: string
  locationId: string
  locationName?: string
  siteId?: string
  siteName?: string
  regionId?: string
  regionName?: string
  status?: string
  uHeight?: number
  position?: string
  powerCapacity?: string
}

export interface Rack extends TypeMeta {
  metadata: ObjectMeta
  spec: RackSpec
}

export interface RackList extends TypeMeta {
  items: Rack[]
  totalCount: number
}

// --- O11y Endpoint ---

export interface EndpointSpec {
  description?: string
  isPublic?: boolean
  metricsUrl?: string
  logsUrl?: string
  tracesUrl?: string
  apmUrl?: string
  status?: "active" | "inactive"
}

export interface Endpoint extends TypeMeta {
  metadata: ObjectMeta
  spec: EndpointSpec
}

export interface EndpointList extends TypeMeta {
  items: Endpoint[]
  totalCount: number
}

// --- Certificate ---

export interface CertificateSpec {
  certType: "ca" | "server" | "client" | "both"
  commonName?: string
  dnsNames?: string[]
  caName?: string
  validityDays?: number
}

export interface CertificateStatus {
  serialNumber: string
  notBefore: string
  notAfter: string
  certificate: string
}

export interface Certificate extends TypeMeta {
  metadata: ObjectMeta
  spec: CertificateSpec
  status?: CertificateStatus
}

export interface CertificateList extends TypeMeta {
  items: Certificate[]
  totalCount: number
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
