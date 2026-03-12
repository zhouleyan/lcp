# Network Module Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Network module frontend (list/detail pages for Networks, Subnets, IP Allocations) following existing IAM/Infra patterns.

**Architecture:** Platform-only module (no workspace/namespace scoping). Three pages: Network list → Network detail (with embedded Subnets table) → Subnet detail (with IP usage bar + embedded Allocations table). All API calls go through `networkApi` client with prefix `/api/network/v1`.

**Tech Stack:** React 19 + TypeScript, shadcn/ui, react-hook-form + zod, ky HTTP client, custom i18n

---

### Task 1: API Types

**Files:**
- Modify: `ui/src/api/types.ts`

**Step 1: Add Network/Subnet/IPAllocation types to types.ts**

Append after the `HostAssignment` types section (before `// --- Infra Requests ---`):

```typescript
// --- Network ---

export interface NetworkSpec {
  displayName?: string
  description?: string
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
  status?: "active" | "inactive"
  networkId?: string
  freeIPs?: number
  usedIPs?: number
  totalIPs?: number
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
```

**Step 2: Commit**

```bash
cd /Users/zhouleyan/Projects/lcp/.worktrees/feat-network
git add ui/src/api/types.ts
git commit -m "feat(ui): add Network/Subnet/IPAllocation API types"
```

---

### Task 2: API Client + Functions

**Files:**
- Create: `ui/src/api/network/client.ts`
- Create: `ui/src/api/network/networks.ts`
- Create: `ui/src/api/network/subnets.ts`
- Create: `ui/src/api/network/allocations.ts`

**Step 1: Create API client**

```typescript
// ui/src/api/network/client.ts
import { api } from "../client"

export const networkApi = api.extend({ prefixUrl: "/api/network/v1" })
```

**Step 2: Create networks API functions**

```typescript
// ui/src/api/network/networks.ts
import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { Network, NetworkList, ListParams } from "../types"

export async function listNetworks(params?: ListParams): Promise<NetworkList> {
  return apiRequest(networkApi.get("networks", { searchParams: params as Record<string, string> }).json())
}

export async function getNetwork(id: string): Promise<Network> {
  return apiRequest(networkApi.get(`networks/${id}`).json())
}

export async function createNetwork(data: Pick<Network, "metadata" | "spec">): Promise<Network> {
  return apiRequest(networkApi.post("networks", { json: data }).json())
}

export async function updateNetwork(id: string, data: Pick<Network, "metadata" | "spec">): Promise<Network> {
  return apiRequest(networkApi.put(`networks/${id}`, { json: data }).json())
}

export async function deleteNetwork(id: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${id}`).json())
}

export async function deleteNetworks(ids: string[]): Promise<void> {
  await apiRequest(networkApi.delete("networks", { json: { ids } }).json())
}
```

**Step 3: Create subnets API functions**

```typescript
// ui/src/api/network/subnets.ts
import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { Subnet, SubnetList, ListParams } from "../types"

export async function listSubnets(networkId: string, params?: ListParams): Promise<SubnetList> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets`, { searchParams: params as Record<string, string> }).json())
}

export async function getSubnet(networkId: string, subnetId: string): Promise<Subnet> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets/${subnetId}`).json())
}

export async function createSubnet(networkId: string, data: Pick<Subnet, "metadata" | "spec">): Promise<Subnet> {
  return apiRequest(networkApi.post(`networks/${networkId}/subnets`, { json: data }).json())
}

export async function updateSubnet(networkId: string, subnetId: string, data: Pick<Subnet, "metadata" | "spec">): Promise<Subnet> {
  return apiRequest(networkApi.put(`networks/${networkId}/subnets/${subnetId}`, { json: data }).json())
}

export async function deleteSubnet(networkId: string, subnetId: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets/${subnetId}`).json())
}

export async function deleteSubnets(networkId: string, ids: string[]): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets`, { json: { ids } }).json())
}
```

**Step 4: Create allocations API functions**

```typescript
// ui/src/api/network/allocations.ts
import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { IPAllocation, IPAllocationList, ListParams } from "../types"

export async function listAllocations(networkId: string, subnetId: string, params?: ListParams): Promise<IPAllocationList> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets/${subnetId}/allocations`, { searchParams: params as Record<string, string> }).json())
}

export async function createAllocation(networkId: string, subnetId: string, data: Pick<IPAllocation, "spec">): Promise<IPAllocation> {
  return apiRequest(networkApi.post(`networks/${networkId}/subnets/${subnetId}/allocations`, { json: data }).json())
}

export async function deleteAllocation(networkId: string, subnetId: string, allocationId: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets/${subnetId}/allocations/${allocationId}`).json())
}
```

**Step 5: Commit**

```bash
git add ui/src/api/network/
git commit -m "feat(ui): add Network module API client and functions"
```

---

### Task 3: i18n Locale Files

**Files:**
- Create: `ui/src/i18n/locales/zh-CN/network.ts`
- Create: `ui/src/i18n/locales/en-US/network.ts`
- Modify: `ui/src/i18n/locales/zh-CN/index.ts`
- Modify: `ui/src/i18n/locales/en-US/index.ts`

**Step 1: Create zh-CN network locale**

```typescript
// ui/src/i18n/locales/zh-CN/network.ts
import type { Messages } from "../../types"

const network: Messages = {
  // network
  "network.title": "网络管理",
  "network.manage": "管理平台网络资源。共 {count} 个。",
  "network.create": "创建网络",
  "network.edit": "编辑网络",
  "network.noData": "暂无网络。",
  "network.name": "网络名称",
  "network.displayName": "显示名称",
  "network.description": "描述",
  "network.status": "状态",
  "network.subnetCount": "子网数量",
  "network.deleteConfirm": "确定要删除网络 \"{name}\" 吗？此操作不可撤销。",
  "network.deleteSelected": "删除选中",
  "network.batchDeleteConfirm": "确定要删除选中的 {count} 个网络吗？此操作不可撤销。",
  "network.searchPlaceholder": "搜索名称、显示名称...",
  "network.detail": "网络详情",
  "network.basicInfo": "基本信息",

  // subnet
  "subnet.title": "子网管理",
  "subnet.manage": "管理网络子网。共 {count} 个。",
  "subnet.create": "创建子网",
  "subnet.edit": "编辑子网",
  "subnet.noData": "暂无子网。",
  "subnet.name": "子网名称",
  "subnet.displayName": "显示名称",
  "subnet.description": "描述",
  "subnet.cidr": "CIDR",
  "subnet.gateway": "网关",
  "subnet.status": "状态",
  "subnet.freeIPs": "可用 IP",
  "subnet.usedIPs": "已用 IP",
  "subnet.totalIPs": "总 IP",
  "subnet.ipUsage": "IP 使用情况",
  "subnet.deleteConfirm": "确定要删除子网 \"{name}\" 吗？此操作不可撤销。",
  "subnet.deleteSelected": "删除选中",
  "subnet.batchDeleteConfirm": "确定要删除选中的 {count} 个子网吗？此操作不可撤销。",
  "subnet.searchPlaceholder": "搜索名称、CIDR...",
  "subnet.detail": "子网详情",
  "subnet.basicInfo": "基本信息",
  "subnet.cidrPlaceholder": "如 10.0.0.0/24",
  "subnet.gatewayPlaceholder": "如 10.0.0.1",

  // allocation
  "allocation.title": "IP 分配",
  "allocation.manage": "管理子网 IP 分配。共 {count} 个。",
  "allocation.create": "分配 IP",
  "allocation.noData": "暂无 IP 分配。",
  "allocation.ip": "IP 地址",
  "allocation.description": "描述",
  "allocation.isGateway": "网关",
  "allocation.deleteConfirm": "确定要释放 IP \"{ip}\" 吗？",
  "allocation.ipPlaceholder": "如 10.0.0.100",
  "allocation.cannotDeleteGateway": "网关 IP 不可直接释放。",

  // nav
  "nav.network": "网络",
  "nav.networks": "网络",

  // permission codes - network
  "perm.group.network": "网络",
  "perm.group.network.networks": "网络",
  "perm.group.network.subnets": "子网",
  "perm.group.network.allocations": "IP 分配",
  "perm.network:networks:list": "查看网络列表",
  "perm.network:networks:get": "查看网络详情",
  "perm.network:networks:create": "创建网络",
  "perm.network:networks:update": "更新网络",
  "perm.network:networks:patch": "修改网络",
  "perm.network:networks:delete": "删除网络",
  "perm.network:networks:deleteCollection": "批量删除网络",
  "perm.network:subnets:list": "查看子网列表",
  "perm.network:subnets:get": "查看子网详情",
  "perm.network:subnets:create": "创建子网",
  "perm.network:subnets:update": "更新子网",
  "perm.network:subnets:patch": "修改子网",
  "perm.network:subnets:delete": "删除子网",
  "perm.network:subnets:deleteCollection": "批量删除子网",
  "perm.network:allocations:list": "查看 IP 分配列表",
  "perm.network:allocations:get": "查看 IP 分配详情",
  "perm.network:allocations:create": "分配 IP",
  "perm.network:allocations:delete": "释放 IP",
}

export default network
```

**Step 2: Create en-US network locale**

```typescript
// ui/src/i18n/locales/en-US/network.ts
import type { Messages } from "../../types"

const network: Messages = {
  // network
  "network.title": "Network Management",
  "network.manage": "Manage platform network resources. {count} total.",
  "network.create": "Create Network",
  "network.edit": "Edit Network",
  "network.noData": "No networks found.",
  "network.name": "Network Name",
  "network.displayName": "Display Name",
  "network.description": "Description",
  "network.status": "Status",
  "network.subnetCount": "Subnet Count",
  "network.deleteConfirm": "Are you sure you want to delete network \"{name}\"? This action cannot be undone.",
  "network.deleteSelected": "Delete Selected",
  "network.batchDeleteConfirm": "Are you sure you want to delete {count} selected networks? This action cannot be undone.",
  "network.searchPlaceholder": "Search name, display name...",
  "network.detail": "Network Detail",
  "network.basicInfo": "Basic Information",

  // subnet
  "subnet.title": "Subnet Management",
  "subnet.manage": "Manage network subnets. {count} total.",
  "subnet.create": "Create Subnet",
  "subnet.edit": "Edit Subnet",
  "subnet.noData": "No subnets found.",
  "subnet.name": "Subnet Name",
  "subnet.displayName": "Display Name",
  "subnet.description": "Description",
  "subnet.cidr": "CIDR",
  "subnet.gateway": "Gateway",
  "subnet.status": "Status",
  "subnet.freeIPs": "Free IPs",
  "subnet.usedIPs": "Used IPs",
  "subnet.totalIPs": "Total IPs",
  "subnet.ipUsage": "IP Usage",
  "subnet.deleteConfirm": "Are you sure you want to delete subnet \"{name}\"? This action cannot be undone.",
  "subnet.deleteSelected": "Delete Selected",
  "subnet.batchDeleteConfirm": "Are you sure you want to delete {count} selected subnets? This action cannot be undone.",
  "subnet.searchPlaceholder": "Search name, CIDR...",
  "subnet.detail": "Subnet Detail",
  "subnet.basicInfo": "Basic Information",
  "subnet.cidrPlaceholder": "e.g. 10.0.0.0/24",
  "subnet.gatewayPlaceholder": "e.g. 10.0.0.1",

  // allocation
  "allocation.title": "IP Allocations",
  "allocation.manage": "Manage subnet IP allocations. {count} total.",
  "allocation.create": "Allocate IP",
  "allocation.noData": "No IP allocations found.",
  "allocation.ip": "IP Address",
  "allocation.description": "Description",
  "allocation.isGateway": "Gateway",
  "allocation.deleteConfirm": "Are you sure you want to release IP \"{ip}\"?",
  "allocation.ipPlaceholder": "e.g. 10.0.0.100",
  "allocation.cannotDeleteGateway": "Cannot directly release gateway IP.",

  // nav
  "nav.network": "Network",
  "nav.networks": "Networks",

  // permission codes - network
  "perm.group.network": "Network",
  "perm.group.network.networks": "Networks",
  "perm.group.network.subnets": "Subnets",
  "perm.group.network.allocations": "IP Allocations",
  "perm.network:networks:list": "List networks",
  "perm.network:networks:get": "Get network details",
  "perm.network:networks:create": "Create network",
  "perm.network:networks:update": "Update network",
  "perm.network:networks:patch": "Patch network",
  "perm.network:networks:delete": "Delete network",
  "perm.network:networks:deleteCollection": "Batch delete networks",
  "perm.network:subnets:list": "List subnets",
  "perm.network:subnets:get": "Get subnet details",
  "perm.network:subnets:create": "Create subnet",
  "perm.network:subnets:update": "Update subnet",
  "perm.network:subnets:patch": "Patch subnet",
  "perm.network:subnets:delete": "Delete subnet",
  "perm.network:subnets:deleteCollection": "Batch delete subnets",
  "perm.network:allocations:list": "List IP allocations",
  "perm.network:allocations:get": "Get IP allocation details",
  "perm.network:allocations:create": "Allocate IP",
  "perm.network:allocations:delete": "Release IP",
}

export default network
```

**Step 3: Update zh-CN/index.ts**

Add `import network from "./network"` and spread `...network` into the object.

**Step 4: Update en-US/index.ts**

Same pattern.

**Step 5: Commit**

```bash
git add ui/src/i18n/locales/
git commit -m "feat(ui): add Network module i18n locale files"
```

---

### Task 4: Module Registration (modules.ts + nav-config.ts + routes.tsx)

**Files:**
- Modify: `ui/src/modules.ts`
- Modify: `ui/src/lib/nav-config.ts`
- Create: `ui/src/pages/network/routes.tsx`
- Modify: `ui/src/routes.tsx`

**Step 1: Add "network" to MODULE_PREFIXES in modules.ts**

Change:
```typescript
export const MODULE_PREFIXES = new Set(["iam", "dashboard", "audit", "infra"])
```
To:
```typescript
export const MODULE_PREFIXES = new Set(["iam", "dashboard", "audit", "infra", "network"])
```

**Step 2: Add networks nav item in nav-config.ts**

Add import `Network` from lucide-react. Add nav item to `NAV_ITEMS` array:

```typescript
{ resource: "networks", module: "network", permission: "network:networks:list", labelKey: "nav.networks", icon: Network, group: "nav.network", scopes: ["platform"] },
```

Note: Network module is platform-only, so `scopes: ["platform"]`.

**Step 3: Create network routes**

```typescript
// ui/src/pages/network/routes.tsx
import { Navigate, type RouteObject } from "react-router"
import NetworkListPage from "./networks/list"
import NetworkDetailPage from "./networks/detail"
import SubnetDetailPage from "./networks/subnet-detail"

export const networkRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/network/networks" replace /> },
  { path: "networks", element: <NetworkListPage /> },
  { path: "networks/:networkId", element: <NetworkDetailPage /> },
  { path: "networks/:networkId/subnets/:subnetId", element: <SubnetDetailPage /> },
]
```

**Step 4: Register network routes in routes.tsx**

Add import:
```typescript
import { networkRoutes } from "@/pages/network/routes"
```

Add to RootLayout children:
```typescript
{
  path: "network",
  children: networkRoutes,
},
```

**Step 5: Commit**

```bash
git add ui/src/modules.ts ui/src/lib/nav-config.ts ui/src/pages/network/routes.tsx ui/src/routes.tsx
git commit -m "feat(ui): register Network module routes and navigation"
```

Note: This commit will fail `pnpm build` because the page components don't exist yet. That's OK — we'll create them in subsequent tasks.

---

### Task 5: Network List Page

**Files:**
- Create: `ui/src/pages/network/networks/list.tsx`

**Step 1: Create NetworkListPage**

Follow the Host list page pattern, simplified for platform-only scope.

Features:
- Search bar (debounced via `useListState`)
- Status filter dropdown on Status column header
- Sortable columns: name, created_at
- Columns: checkbox, name (link to detail), displayName, status, subnetCount, createdAt, actions
- Actions dropdown: edit, delete
- Batch delete
- Create dialog (name + displayName + description + status)
- Edit dialog (same fields, name disabled)
- Permissions: `network:networks:create`, `network:networks:update`, `network:networks:delete`, `network:networks:deleteCollection`

The create/edit form uses a `NetworkFormDialog` component embedded in the same file (matching infra pattern).

Zod schema for form:
```typescript
const schema = z.object({
  name: z.string()
    .min(3, t("api.validation.name.format"))
    .max(50, t("api.validation.name.format"))
    .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
  displayName: z.string().optional(),
  description: z.string().optional(),
  status: z.enum(["active", "inactive"]),
})
```

**Step 2: Commit**

```bash
git add ui/src/pages/network/networks/list.tsx
git commit -m "feat(ui): add Network list page with CRUD"
```

---

### Task 6: Network Detail Page (with Subnets table)

**Files:**
- Create: `ui/src/pages/network/networks/detail.tsx`

**Step 1: Create NetworkDetailPage**

Structure:
1. Header: network name + status badge + edit/delete buttons
2. Basic info card (name, displayName, description, status, subnetCount, createdAt, updatedAt)
3. Subnets section: full-featured table embedded in Card
   - Search, status filter, sortable columns
   - Columns: checkbox, name (link to subnet detail), cidr, gateway, freeIPs/usedIPs/totalIPs, status, createdAt, actions
   - Create Subnet dialog: name + displayName + description + cidr + gateway + status
   - Subnet actions: edit, delete
   - Batch delete subnets

Subnet create form zod schema:
```typescript
const schema = z.object({
  name: z.string()
    .min(3, t("api.validation.name.format"))
    .max(50, t("api.validation.name.format"))
    .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
  displayName: z.string().optional(),
  description: z.string().optional(),
  cidr: z.string().min(1, t("api.validation.required", { field: t("subnet.cidr") })),
  gateway: z.string().optional(),
  status: z.enum(["active", "inactive"]),
})
```

Link to subnet detail: `<Link to={`subnets/${subnet.metadata.id}`}>` (relative path).

Permissions: `network:subnets:create`, `network:subnets:update`, `network:subnets:delete`, `network:subnets:deleteCollection`

Edit network dialog follows same pattern as list page.

**Step 2: Commit**

```bash
git add ui/src/pages/network/networks/detail.tsx
git commit -m "feat(ui): add Network detail page with Subnets table"
```

---

### Task 7: Subnet Detail Page (with IP Allocations table)

**Files:**
- Create: `ui/src/pages/network/networks/subnet-detail.tsx`

**Step 1: Create SubnetDetailPage**

Structure:
1. Header: subnet name + status badge + edit/delete buttons
2. IP Usage card: progress bar showing usedIPs/totalIPs
   - Visual: `<div className="h-2 rounded-full bg-muted"><div className="h-2 rounded-full bg-primary" style={{ width: `${percentage}%` }} /></div>`
   - Text: "usedIPs / totalIPs used (freeIPs available)"
3. Basic info card (name, displayName, description, cidr, gateway, status, networkId, createdAt, updatedAt)
4. Allocations section: table embedded in Card
   - Sortable columns: ip, created_at
   - Columns: ip, description, isGateway badge, createdAt, actions (delete button, disabled if isGateway)
   - Create Allocation dialog: ip + description
   - Delete allocation with confirm dialog

Allocation create form zod schema:
```typescript
const schema = z.object({
  ip: z.string().min(1, t("api.validation.required", { field: t("allocation.ip") })),
  description: z.string().optional(),
})
```

Permissions: `network:allocations:create`, `network:allocations:delete`

Gateway rows: show `isGateway` badge, delete button disabled with tooltip "网关 IP 不可直接释放".

Edit subnet dialog: similar to subnet create but name/cidr disabled.

**Step 2: Commit**

```bash
git add ui/src/pages/network/networks/subnet-detail.tsx
git commit -m "feat(ui): add Subnet detail page with IP Allocations table"
```

---

### Task 8: Error Translation Mappings

**Files:**
- Modify: `ui/src/api/client.ts`

**Step 1: Add network-specific error message mappings**

Add to `messagePrefixMap`:
```typescript
"cannot delete network": "api.error.cannotDeleteNetwork",
"cannot delete subnet": "api.error.cannotDeleteSubnet",
"cannot delete gateway IP allocation": "api.error.cannotDeleteGateway",
```

**Step 2: Add i18n keys to both locale common files**

Add to `zh-CN/common.ts`:
```typescript
"api.error.cannotDeleteNetwork": "无法删除网络：存在关联子网",
"api.error.cannotDeleteSubnet": "无法删除子网：存在非网关 IP 分配",
"api.error.cannotDeleteGateway": "网关 IP 不可直接释放",
```

Add to `en-US/common.ts`:
```typescript
"api.error.cannotDeleteNetwork": "Cannot delete network: has associated subnets",
"api.error.cannotDeleteSubnet": "Cannot delete subnet: has non-gateway IP allocations",
"api.error.cannotDeleteGateway": "Cannot directly release gateway IP allocation",
```

**Step 3: Commit**

```bash
git add ui/src/api/client.ts ui/src/i18n/locales/zh-CN/common.ts ui/src/i18n/locales/en-US/common.ts
git commit -m "feat(ui): add Network module error translation mappings"
```

---

### Task 9: Type Check + Build Verification

**Step 1: Run type check**

```bash
cd /Users/zhouleyan/Projects/lcp/.worktrees/feat-network/ui
npx tsc --noEmit
```

**Step 2: Fix any type errors**

**Step 3: Run build**

```bash
pnpm build
```

**Step 4: Commit fixes if any**

```bash
git add -A
git commit -m "fix(ui): resolve Network module type/build errors"
```
