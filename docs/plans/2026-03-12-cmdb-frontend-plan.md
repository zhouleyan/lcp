# CMDB Frontend Implementation Plan: Region / Site / Location Pages

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build frontend pages for three platform-only CMDB resources (Region, Site, Location) following the existing Infra module patterns.

**Architecture:** Platform-only resources with no workspace/namespace scope. Direct API calls (no `scopedApiCall`). Region → Site → Location hierarchy with parent detail pages showing child sub-tables. All pages share the established list/detail/form dialog patterns from Environment/Host pages.

**Tech Stack:** React 19, TypeScript, shadcn/ui, ky, react-hook-form + zod/v4, lucide-react icons

---

### Task 1: Add TypeScript Types

**Files:**
- Modify: `ui/src/api/types.ts`

**Step 1: Add Region, Site, Location types at the end of the file (before StatusResponse)**

Append these types after the `BindEnvironmentRequest` interface (after line 350) and before `StatusResponseDetail`:

```typescript
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
```

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors related to Region/Site/Location types

**Step 3: Commit**

```bash
git add ui/src/api/types.ts
git commit -m "feat(ui): add Region/Site/Location TypeScript types"
```

---

### Task 2: Create API Client Files

**Files:**
- Create: `ui/src/api/infra/regions.ts`
- Create: `ui/src/api/infra/sites.ts`
- Create: `ui/src/api/infra/locations.ts`

These resources are platform-only — no workspace/namespace API variants needed.

**Step 1: Create `ui/src/api/infra/regions.ts`**

```typescript
import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Region, RegionList, SiteList, ListParams } from "../types"

export async function listRegions(params?: ListParams): Promise<RegionList> {
  return apiRequest(infraApi.get("regions", { searchParams: params as Record<string, string> }).json())
}

export async function getRegion(id: string): Promise<Region> {
  return apiRequest(infraApi.get(`regions/${id}`).json())
}

export async function createRegion(data: Pick<Region, "metadata" | "spec">): Promise<Region> {
  return apiRequest(infraApi.post("regions", { json: data }).json())
}

export async function updateRegion(id: string, data: Pick<Region, "metadata" | "spec">): Promise<Region> {
  return apiRequest(infraApi.put(`regions/${id}`, { json: data }).json())
}

export async function patchRegion(
  id: string,
  data: Partial<Pick<Region, "metadata" | "spec">>,
): Promise<Region> {
  return apiRequest(infraApi.patch(`regions/${id}`, { json: data }).json())
}

export async function deleteRegion(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`regions/${id}`).json())
}

export async function deleteRegions(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("regions", { json: { ids } }).json())
}

export async function getRegionSites(id: string, params?: ListParams): Promise<SiteList> {
  return apiRequest(
    infraApi.get(`regions/${id}/sites`, { searchParams: params as Record<string, string> }).json(),
  )
}
```

**Step 2: Create `ui/src/api/infra/sites.ts`**

```typescript
import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Site, SiteList, LocationList, ListParams } from "../types"

export async function listSites(params?: ListParams): Promise<SiteList> {
  return apiRequest(infraApi.get("sites", { searchParams: params as Record<string, string> }).json())
}

export async function getSite(id: string): Promise<Site> {
  return apiRequest(infraApi.get(`sites/${id}`).json())
}

export async function createSite(data: Pick<Site, "metadata" | "spec">): Promise<Site> {
  return apiRequest(infraApi.post("sites", { json: data }).json())
}

export async function updateSite(id: string, data: Pick<Site, "metadata" | "spec">): Promise<Site> {
  return apiRequest(infraApi.put(`sites/${id}`, { json: data }).json())
}

export async function patchSite(
  id: string,
  data: Partial<Pick<Site, "metadata" | "spec">>,
): Promise<Site> {
  return apiRequest(infraApi.patch(`sites/${id}`, { json: data }).json())
}

export async function deleteSite(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`sites/${id}`).json())
}

export async function deleteSites(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("sites", { json: { ids } }).json())
}

export async function getSiteLocations(id: string, params?: ListParams): Promise<LocationList> {
  return apiRequest(
    infraApi.get(`sites/${id}/locations`, { searchParams: params as Record<string, string> }).json(),
  )
}
```

**Step 3: Create `ui/src/api/infra/locations.ts`**

```typescript
import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Location, LocationList, ListParams } from "../types"

export async function listLocations(params?: ListParams): Promise<LocationList> {
  return apiRequest(infraApi.get("locations", { searchParams: params as Record<string, string> }).json())
}

export async function getLocation(id: string): Promise<Location> {
  return apiRequest(infraApi.get(`locations/${id}`).json())
}

export async function createLocation(data: Pick<Location, "metadata" | "spec">): Promise<Location> {
  return apiRequest(infraApi.post("locations", { json: data }).json())
}

export async function updateLocation(
  id: string,
  data: Pick<Location, "metadata" | "spec">,
): Promise<Location> {
  return apiRequest(infraApi.put(`locations/${id}`, { json: data }).json())
}

export async function patchLocation(
  id: string,
  data: Partial<Pick<Location, "metadata" | "spec">>,
): Promise<Location> {
  return apiRequest(infraApi.patch(`locations/${id}`, { json: data }).json())
}

export async function deleteLocation(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`locations/${id}`).json())
}

export async function deleteLocations(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("locations", { json: { ids } }).json())
}
```

**Step 4: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

**Step 5: Commit**

```bash
git add ui/src/api/infra/regions.ts ui/src/api/infra/sites.ts ui/src/api/infra/locations.ts
git commit -m "feat(ui): add Region/Site/Location API client functions"
```

---

### Task 3: Add i18n Translations

**Files:**
- Modify: `ui/src/i18n/locales/en-US/common.ts` (nav keys)
- Modify: `ui/src/i18n/locales/zh-CN/common.ts` (nav keys)
- Modify: `ui/src/i18n/locales/en-US/infra.ts` (resource translations)
- Modify: `ui/src/i18n/locales/zh-CN/infra.ts` (resource translations)

**Step 1: Add nav keys to `en-US/common.ts`**

After line 57 (`"nav.environments": "Environments",`), add:

```typescript
  "nav.regions": "Regions",
  "nav.sites": "Sites",
  "nav.locations": "Locations",
```

**Step 2: Add nav keys to `zh-CN/common.ts`**

After line 57 (`"nav.environments": "环境",`), add:

```typescript
  "nav.regions": "区域",
  "nav.sites": "站点",
  "nav.locations": "机房",
```

**Step 3: Add resource translations to `en-US/infra.ts`**

Append before `}` at the end of the `infra` object, after the environment permission codes:

```typescript
  // region
  "region.title": "Region Management",
  "region.manage": "Manage regions (availability zones). {count} total.",
  "region.create": "Create Region",
  "region.edit": "Edit Region",
  "region.noData": "No regions found.",
  "region.name": "Region Name",
  "region.displayName": "Display Name",
  "region.description": "Description",
  "region.status": "Status",
  "region.latitude": "Latitude",
  "region.longitude": "Longitude",
  "region.siteCount": "Site Count",
  "region.sites": "Sites in Region",
  "region.sitesEmpty": "No sites in this region.",
  "region.deleteConfirm": "Are you sure you want to delete region \"{name}\"? This region must have no child sites.",
  "region.deleteSelected": "Delete Selected",
  "region.batchDeleteConfirm": "Are you sure you want to delete {count} selected regions? Regions with child sites cannot be deleted.",
  "region.searchPlaceholder": "Search name, display name...",
  "region.detail": "Region Detail",
  "region.basicInfo": "Basic Information",

  // site
  "site.title": "Site Management",
  "site.manage": "Manage data center sites. {count} total.",
  "site.create": "Create Site",
  "site.edit": "Edit Site",
  "site.noData": "No sites found.",
  "site.name": "Site Name",
  "site.displayName": "Display Name",
  "site.description": "Description",
  "site.regionId": "Region",
  "site.regionName": "Region",
  "site.status": "Status",
  "site.address": "Address",
  "site.latitude": "Latitude",
  "site.longitude": "Longitude",
  "site.contactName": "Contact Name",
  "site.contactPhone": "Contact Phone",
  "site.contactEmail": "Contact Email",
  "site.locationCount": "Location Count",
  "site.locations": "Locations in Site",
  "site.locationsEmpty": "No locations in this site.",
  "site.deleteConfirm": "Are you sure you want to delete site \"{name}\"? This site must have no child locations.",
  "site.deleteSelected": "Delete Selected",
  "site.batchDeleteConfirm": "Are you sure you want to delete {count} selected sites? Sites with child locations cannot be deleted.",
  "site.searchPlaceholder": "Search name, display name...",
  "site.filter.region": "Region",
  "site.filter.regionAll": "All Regions",
  "site.detail": "Site Detail",
  "site.basicInfo": "Basic Information",
  "site.contactInfo": "Contact Information",
  "site.selectRegion": "Select Region",

  // location
  "location.title": "Location Management",
  "location.manage": "Manage data center locations. {count} total.",
  "location.create": "Create Location",
  "location.edit": "Edit Location",
  "location.noData": "No locations found.",
  "location.name": "Location Name",
  "location.displayName": "Display Name",
  "location.description": "Description",
  "location.siteId": "Site",
  "location.siteName": "Site",
  "location.regionName": "Region",
  "location.status": "Status",
  "location.floor": "Floor",
  "location.rackCapacity": "Rack Capacity",
  "location.contactName": "Contact Name",
  "location.contactPhone": "Contact Phone",
  "location.contactEmail": "Contact Email",
  "location.deleteConfirm": "Are you sure you want to delete location \"{name}\"? This action cannot be undone.",
  "location.deleteSelected": "Delete Selected",
  "location.batchDeleteConfirm": "Are you sure you want to delete {count} selected locations? This action cannot be undone.",
  "location.searchPlaceholder": "Search name, display name...",
  "location.filter.site": "Site",
  "location.filter.siteAll": "All Sites",
  "location.filter.region": "Region",
  "location.filter.regionAll": "All Regions",
  "location.detail": "Location Detail",
  "location.basicInfo": "Basic Information",
  "location.contactInfo": "Contact Information",
  "location.selectSite": "Select Site",

  // permission codes - region/site/location
  "perm.group.infra.regions": "Regions",
  "perm.group.infra.sites": "Sites",
  "perm.group.infra.locations": "Locations",
  "perm.infra:regions:list": "List regions",
  "perm.infra:regions:get": "Get region details",
  "perm.infra:regions:create": "Create region",
  "perm.infra:regions:update": "Update region",
  "perm.infra:regions:patch": "Patch region",
  "perm.infra:regions:delete": "Delete region",
  "perm.infra:regions:deleteCollection": "Batch delete regions",
  "perm.infra:sites:list": "List sites",
  "perm.infra:sites:get": "Get site details",
  "perm.infra:sites:create": "Create site",
  "perm.infra:sites:update": "Update site",
  "perm.infra:sites:patch": "Patch site",
  "perm.infra:sites:delete": "Delete site",
  "perm.infra:sites:deleteCollection": "Batch delete sites",
  "perm.infra:locations:list": "List locations",
  "perm.infra:locations:get": "Get location details",
  "perm.infra:locations:create": "Create location",
  "perm.infra:locations:update": "Update location",
  "perm.infra:locations:patch": "Patch location",
  "perm.infra:locations:delete": "Delete location",
  "perm.infra:locations:deleteCollection": "Batch delete locations",
```

**Step 4: Add resource translations to `zh-CN/infra.ts`**

Append the same set of keys with Chinese translations:

```typescript
  // region
  "region.title": "区域管理",
  "region.manage": "管理可用域/地理区域。共 {count} 个。",
  "region.create": "创建区域",
  "region.edit": "编辑区域",
  "region.noData": "暂无区域。",
  "region.name": "区域名称",
  "region.displayName": "显示名称",
  "region.description": "描述",
  "region.status": "状态",
  "region.latitude": "纬度",
  "region.longitude": "经度",
  "region.siteCount": "站点数量",
  "region.sites": "下属站点",
  "region.sitesEmpty": "该区域下暂无站点。",
  "region.deleteConfirm": "确定要删除区域 \"{name}\" 吗？该区域下不能有站点。",
  "region.deleteSelected": "删除选中",
  "region.batchDeleteConfirm": "确定要删除选中的 {count} 个区域吗？包含站点的区域无法删除。",
  "region.searchPlaceholder": "搜索名称、显示名称...",
  "region.detail": "区域详情",
  "region.basicInfo": "基本信息",

  // site
  "site.title": "站点管理",
  "site.manage": "管理数据中心站点。共 {count} 个。",
  "site.create": "创建站点",
  "site.edit": "编辑站点",
  "site.noData": "暂无站点。",
  "site.name": "站点名称",
  "site.displayName": "显示名称",
  "site.description": "描述",
  "site.regionId": "所属区域",
  "site.regionName": "所属区域",
  "site.status": "状态",
  "site.address": "地址",
  "site.latitude": "纬度",
  "site.longitude": "经度",
  "site.contactName": "负责人",
  "site.contactPhone": "联系电话",
  "site.contactEmail": "联系邮箱",
  "site.locationCount": "机房数量",
  "site.locations": "下属机房",
  "site.locationsEmpty": "该站点下暂无机房。",
  "site.deleteConfirm": "确定要删除站点 \"{name}\" 吗？该站点下不能有机房。",
  "site.deleteSelected": "删除选中",
  "site.batchDeleteConfirm": "确定要删除选中的 {count} 个站点吗？包含机房的站点无法删除。",
  "site.searchPlaceholder": "搜索名称、显示名称...",
  "site.filter.region": "所属区域",
  "site.filter.regionAll": "全部区域",
  "site.detail": "站点详情",
  "site.basicInfo": "基本信息",
  "site.contactInfo": "联系信息",
  "site.selectRegion": "选择区域",

  // location
  "location.title": "机房管理",
  "location.manage": "管理数据中心机房。共 {count} 个。",
  "location.create": "创建机房",
  "location.edit": "编辑机房",
  "location.noData": "暂无机房。",
  "location.name": "机房名称",
  "location.displayName": "显示名称",
  "location.description": "描述",
  "location.siteId": "所属站点",
  "location.siteName": "所属站点",
  "location.regionName": "所属区域",
  "location.status": "状态",
  "location.floor": "楼层",
  "location.rackCapacity": "机柜容量",
  "location.contactName": "负责人",
  "location.contactPhone": "联系电话",
  "location.contactEmail": "联系邮箱",
  "location.deleteConfirm": "确定要删除机房 \"{name}\" 吗？此操作不可撤销。",
  "location.deleteSelected": "删除选中",
  "location.batchDeleteConfirm": "确定要删除选中的 {count} 个机房吗？此操作不可撤销。",
  "location.searchPlaceholder": "搜索名称、显示名称...",
  "location.filter.site": "所属站点",
  "location.filter.siteAll": "全部站点",
  "location.filter.region": "所属区域",
  "location.filter.regionAll": "全部区域",
  "location.detail": "机房详情",
  "location.basicInfo": "基本信息",
  "location.contactInfo": "联系信息",
  "location.selectSite": "选择站点",

  // permission codes - region/site/location
  "perm.group.infra.regions": "区域",
  "perm.group.infra.sites": "站点",
  "perm.group.infra.locations": "机房",
  "perm.infra:regions:list": "查看区域列表",
  "perm.infra:regions:get": "查看区域详情",
  "perm.infra:regions:create": "创建区域",
  "perm.infra:regions:update": "更新区域",
  "perm.infra:regions:patch": "修改区域",
  "perm.infra:regions:delete": "删除区域",
  "perm.infra:regions:deleteCollection": "批量删除区域",
  "perm.infra:sites:list": "查看站点列表",
  "perm.infra:sites:get": "查看站点详情",
  "perm.infra:sites:create": "创建站点",
  "perm.infra:sites:update": "更新站点",
  "perm.infra:sites:patch": "修改站点",
  "perm.infra:sites:delete": "删除站点",
  "perm.infra:sites:deleteCollection": "批量删除站点",
  "perm.infra:locations:list": "查看机房列表",
  "perm.infra:locations:get": "查看机房详情",
  "perm.infra:locations:create": "创建机房",
  "perm.infra:locations:update": "更新机房",
  "perm.infra:locations:patch": "修改机房",
  "perm.infra:locations:delete": "删除机房",
  "perm.infra:locations:deleteCollection": "批量删除机房",
```

**Step 5: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors (i18n Messages type must match across locales)

**Step 6: Commit**

```bash
git add ui/src/i18n/locales/en-US/common.ts ui/src/i18n/locales/zh-CN/common.ts ui/src/i18n/locales/en-US/infra.ts ui/src/i18n/locales/zh-CN/infra.ts
git commit -m "feat(ui): add i18n translations for Region/Site/Location"
```

---

### Task 4: Update Navigation Config

**Files:**
- Modify: `ui/src/lib/nav-config.ts`

**Step 1: Add lucide-react icon imports**

Add `MapPin, Building2 as Building2Icon, Warehouse` to the import from `lucide-react`. Note: `Building2` is already imported; alias the new one or reuse it. Check the existing import — `Building2` is already used for workspaces. Per the design doc:
- Regions → `MapPin`
- Sites → `Building2` (already imported, reuse it — but it conflicts with workspaces). Use a different icon or alias. The design doc says `Building2` for Sites. Since `Building2` is already imported for workspaces, we can just reference it for both — lucide icons are just components. But to avoid confusion, let's just reference the same import.
- Locations → `Warehouse`

Add `MapPin` and `Warehouse` to the existing import:

```typescript
import {
  Home,
  Users,
  Building2,
  FolderKanban,
  Shield,
  ShieldCheck,
  Server,
  Layers,
  ScrollText,
  MapPin,
  Warehouse,
} from "lucide-react"
```

**Step 2: Add NAV_ITEMS entries**

After the `environments` entry (line 45) and before the `logs` entry, add:

```typescript
  { resource: "regions", module: "infra", permission: "infra:regions:list", labelKey: "nav.regions", icon: MapPin, group: "nav.infra", scopes: ["platform"] },
  { resource: "sites", module: "infra", permission: "infra:sites:list", labelKey: "nav.sites", icon: Building2, group: "nav.infra", scopes: ["platform"] },
  { resource: "locations", module: "infra", permission: "infra:locations:list", labelKey: "nav.locations", icon: Warehouse, group: "nav.infra", scopes: ["platform"] },
```

Key difference: `scopes: ["platform"]` — these resources only appear in platform-level sidebar, not workspace/namespace.

**Step 3: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

**Step 4: Commit**

```bash
git add ui/src/lib/nav-config.ts
git commit -m "feat(ui): add Region/Site/Location to navigation config"
```

---

### Task 5: Update Routes

**Files:**
- Modify: `ui/src/pages/infra/routes.tsx`

**Step 1: Add imports and route entries**

Platform-only routes — no workspace/namespace variants needed.

Add imports at the top:

```typescript
import RegionListPage from "./regions/list"
import RegionDetailPage from "./regions/detail"
import SiteListPage from "./sites/list"
import SiteDetailPage from "./sites/detail"
import LocationListPage from "./locations/list"
import LocationDetailPage from "./locations/detail"
```

Add route entries after the existing namespace-level environment routes (before the closing `]`):

```typescript
  // CMDB - Platform-only
  { path: "regions", element: <RegionListPage /> },
  { path: "regions/:regionId", element: <RegionDetailPage /> },
  { path: "sites", element: <SiteListPage /> },
  { path: "sites/:siteId", element: <SiteDetailPage /> },
  { path: "locations", element: <LocationListPage /> },
  { path: "locations/:locationId", element: <LocationDetailPage /> },
```

**Note:** These routes will cause TS errors until the page components are created in Tasks 6-11. That's expected — each subsequent task will resolve them one by one. If you want to avoid intermediate compile errors, create stub files first:

```bash
mkdir -p ui/src/pages/infra/regions ui/src/pages/infra/sites ui/src/pages/infra/locations
```

Create minimal stubs for each page:
- `ui/src/pages/infra/regions/list.tsx` → `export default function RegionListPage() { return <div>TODO</div> }`
- `ui/src/pages/infra/regions/detail.tsx` → `export default function RegionDetailPage() { return <div>TODO</div> }`
- `ui/src/pages/infra/sites/list.tsx` → same pattern
- `ui/src/pages/infra/sites/detail.tsx` → same pattern
- `ui/src/pages/infra/locations/list.tsx` → same pattern
- `ui/src/pages/infra/locations/detail.tsx` → same pattern

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors (if stubs are created)

**Step 3: Commit**

```bash
git add ui/src/pages/infra/routes.tsx ui/src/pages/infra/regions/ ui/src/pages/infra/sites/ ui/src/pages/infra/locations/
git commit -m "feat(ui): add routes and page stubs for Region/Site/Location"
```

---

### Task 6: Region List Page

**Files:**
- Create: `ui/src/pages/infra/regions/list.tsx`

**Context:** Follow `ui/src/pages/infra/environments/list.tsx` pattern. Simplified: no scope-aware routing, no `scopedApiCall`, no `useParams` for workspaceId/namespaceId.

**Step 1: Implement the full Region list page**

The page includes:
- Search by name/displayName
- Status filter dropdown on status column header
- Sortable columns: name, created_at
- Table columns: checkbox, name (Link to detail), displayName, status, siteCount, createdAt, actions
- Create/Edit form dialog (inline component `RegionFormDialog`)
- Delete/batch delete with `ConfirmDialog`
- Permission checks: `infra:regions:create`, `infra:regions:update`, `infra:regions:delete`, `infra:regions:deleteCollection`

Form fields for create/edit:
- `name` (string, required, regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$`, disabled on edit)
- `displayName` (string, optional)
- `description` (string/textarea, optional)
- `status` (select: active/inactive, default "active")
- `latitude` (number, optional)
- `longitude` (number, optional)

Key imports:
```typescript
import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router"
import { Plus, Pencil, Trash2, Search } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { listRegions, createRegion, updateRegion, deleteRegion, deleteRegions } from "@/api/infra/regions"
import { showApiError } from "@/api/client"
import type { Region, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"
```

Permission prefix: `"infra:regions"`
No `permScope` needed (platform-only, pass `undefined`).

Date formatting: use `new Date(item.metadata.createdAt).toLocaleDateString()` — same pattern as environments list.

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`
Expected: No errors

**Step 3: Commit**

```bash
git add ui/src/pages/infra/regions/list.tsx
git commit -m "feat(ui): implement Region list page"
```

---

### Task 7: Region Detail Page

**Files:**
- Create: `ui/src/pages/infra/regions/detail.tsx`

**Context:** Follow `ui/src/pages/infra/environments/detail.tsx` pattern. Region detail shows:
- Header: name + status badge + edit/delete buttons
- Overview card: site count (MapPin icon)
- Basic info card: name, displayName, description, status, latitude, longitude, createdAt, updatedAt
- Sub-resource table: Sites in this region (fetched via `getRegionSites`)

Sites sub-table columns: name (Link to `/infra/sites/:siteId`), displayName, status, locationCount, createdAt. Includes pagination.

Key imports:
```typescript
import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2, MapPin } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { getRegion, getRegionSites, deleteRegion } from "@/api/infra/regions"
import { updateRegion } from "@/api/infra/regions"
import { showApiError } from "@/api/client"
import type { Region, Site, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"
```

Uses `useParams()` to get `regionId`. The edit functionality reuses `RegionFormDialog` — import it from `./list` or define it inline. **Preferred approach**: define a shared `RegionFormDialog` within the list file and export it, then import in detail. OR duplicate the dialog in the detail page (simpler, follows existing Environment pattern where the edit dialog is duplicated in detail.tsx).

Follow the Environment pattern: duplicate the form dialog as `EditRegionDialog` in the detail page (simpler, no cross-file dependencies).

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`

**Step 3: Commit**

```bash
git add ui/src/pages/infra/regions/detail.tsx
git commit -m "feat(ui): implement Region detail page with Sites sub-table"
```

---

### Task 8: Site List Page

**Files:**
- Create: `ui/src/pages/infra/sites/list.tsx`

**Context:** Same pattern as Region list, with additional features:
- **Region filter dropdown** (besides status filter): Fetch regions list on mount for the dropdown. Filter by `regionId` param.
- Table columns: checkbox, name (Link), displayName, regionName (with Link to `/infra/regions/:regionId`), status, locationCount, createdAt, actions

Additional state needed:
```typescript
const [regionFilter, setRegionFilter] = useState("all")
const [allRegions, setAllRegions] = useState<Region[]>([])
```

Fetch all regions for the filter dropdown on mount:
```typescript
useEffect(() => {
  listRegions({ page: 1, pageSize: 200 }).then(data => setAllRegions(data.items ?? [])).catch(() => {})
}, [])
```

Pass `regionId` to list params:
```typescript
if (regionFilter !== "all") params.regionId = regionFilter
```

Form fields for create/edit:
- `name` (required, regex, disabled on edit)
- `displayName` (optional)
- `description` (textarea, optional)
- `regionId` (Select from fetched regions list, required, disabled on edit)
- `status` (select: active/inactive, default "active")
- `address` (optional)
- `latitude` (number, optional)
- `longitude` (number, optional)
- `contactName` (optional)
- `contactPhone` (optional)
- `contactEmail` (optional)

Permission prefix: `"infra:sites"`

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`

**Step 3: Commit**

```bash
git add ui/src/pages/infra/sites/list.tsx
git commit -m "feat(ui): implement Site list page with region filter"
```

---

### Task 9: Site Detail Page

**Files:**
- Create: `ui/src/pages/infra/sites/detail.tsx`

**Context:** Similar to Region detail. Site detail shows:
- Header: name + status badge + edit/delete buttons
- Overview card: location count (Warehouse icon)
- Basic info card: name, displayName, description, regionName (Link to region detail), status, address, latitude, longitude, createdAt, updatedAt
- Contact info card: contactName, contactPhone, contactEmail
- Sub-resource table: Locations in this site (fetched via `getSiteLocations`)

Locations sub-table columns: name (Link to `/infra/locations/:locationId`), displayName, floor, rackCapacity, status, createdAt. Includes pagination.

Uses `useParams()` to get `siteId`.

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`

**Step 3: Commit**

```bash
git add ui/src/pages/infra/sites/detail.tsx
git commit -m "feat(ui): implement Site detail page with Locations sub-table"
```

---

### Task 10: Location List Page

**Files:**
- Create: `ui/src/pages/infra/locations/list.tsx`

**Context:** Same pattern as Site list, with two filter dropdowns:
- **Region filter**: Fetch regions for dropdown, filter by `regionId` param
- **Site filter**: Fetch sites (optionally filtered by selected region) for dropdown, filter by `siteId` param

When region filter changes, reset site filter to "all" and re-fetch sites for that region.

Additional state:
```typescript
const [regionFilter, setRegionFilter] = useState("all")
const [siteFilter, setSiteFilter] = useState("all")
const [allRegions, setAllRegions] = useState<Region[]>([])
const [allSites, setAllSites] = useState<Site[]>([])
```

Fetch sites when region filter changes:
```typescript
useEffect(() => {
  const params: ListParams = { page: 1, pageSize: 200 }
  if (regionFilter !== "all") params.regionId = regionFilter
  listSites(params).then(data => setAllSites(data.items ?? [])).catch(() => {})
}, [regionFilter])
```

Table columns: checkbox, name (Link), displayName, siteName (Link to `/infra/sites/:siteId`), regionName (Link to `/infra/regions/:regionId`), floor, rackCapacity, status, createdAt, actions

Form fields for create/edit:
- `name` (required, regex, disabled on edit)
- `displayName` (optional)
- `description` (textarea, optional)
- `siteId` (Select from fetched sites list, required, disabled on edit)
- `status` (select: active/inactive, default "active")
- `floor` (optional)
- `rackCapacity` (number, optional, min 0)
- `contactName` (optional)
- `contactPhone` (optional)
- `contactEmail` (optional)

Permission prefix: `"infra:locations"`

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`

**Step 3: Commit**

```bash
git add ui/src/pages/infra/locations/list.tsx
git commit -m "feat(ui): implement Location list page with region/site filters"
```

---

### Task 11: Location Detail Page

**Files:**
- Create: `ui/src/pages/infra/locations/detail.tsx`

**Context:** Simplest detail page — no sub-resource table (Location is the leaf resource).

Location detail shows:
- Header: name + status badge + edit/delete buttons
- Basic info card: name, displayName, description, siteName (Link to `/infra/sites/:siteId`), regionName (Link to `/infra/regions/:regionId`), status, floor, rackCapacity, createdAt, updatedAt
- Contact info card: contactName, contactPhone, contactEmail

Uses `useParams()` to get `locationId`. No sub-resource table needed.

**Step 2: Verify types compile**

Run: `cd ui && npx tsc --noEmit --pretty 2>&1 | head -20`

**Step 3: Commit**

```bash
git add ui/src/pages/infra/locations/detail.tsx
git commit -m "feat(ui): implement Location detail page"
```

---

### Task 12: Final Verification

**Step 1: Full type check**

Run: `cd ui && npx tsc --noEmit --pretty`
Expected: No errors

**Step 2: Build check**

Run: `cd ui && pnpm build`
Expected: Build succeeds

**Step 3: Commit any fixes if needed**

If there are type errors or build issues, fix and commit.
