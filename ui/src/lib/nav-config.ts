import {
  Home,
  Users,
  Building2,
  FolderKanban,
  Shield,
  ShieldCheck,
  Server,
  Layers,
  MapPin,
  Warehouse,
  ScrollText,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"

export type ScopeLevel = "platform" | "workspace" | "namespace"

export interface NavItemConfig {
  /** URL segment: "hosts", "users", etc. */
  resource: string
  /** Module prefix: "iam", "infra", "dashboard", "audit" */
  module: string
  /** Permission code for list access */
  permission: string
  /** i18n key for nav label */
  labelKey: string
  /** Lucide icon component */
  icon: LucideIcon
  /** Nav group label key (omit for standalone items like overview) */
  group?: string
  /** Which scope levels this item appears in the sidebar */
  scopes: ScopeLevel[]
}

/**
 * Single source of truth for all navigable resources.
 * Used by: sidebar nav, scope selector, permission checks, breadcrumbs.
 */
export const NAV_ITEMS: NavItemConfig[] = [
  { resource: "overview", module: "dashboard", permission: "dashboard:overview:list", labelKey: "nav.overview", icon: Home, scopes: ["platform", "workspace", "namespace"] },
  { resource: "workspaces", module: "iam", permission: "iam:workspaces:list", labelKey: "nav.workspaces", icon: Building2, group: "nav.iam", scopes: ["platform"] },
  { resource: "namespaces", module: "iam", permission: "iam:namespaces:list", labelKey: "nav.namespaces", icon: FolderKanban, group: "nav.iam", scopes: ["platform", "workspace"] },
  { resource: "users", module: "iam", permission: "iam:users:list", labelKey: "nav.users", icon: Users, group: "nav.iam", scopes: ["platform", "workspace", "namespace"] },
  { resource: "roles", module: "iam", permission: "iam:roles:list", labelKey: "nav.roles", icon: Shield, group: "nav.iam", scopes: ["platform", "workspace", "namespace"] },
  { resource: "rolebindings", module: "iam", permission: "iam:rolebindings:list", labelKey: "nav.rolebindings", icon: ShieldCheck, group: "nav.iam", scopes: ["platform", "workspace", "namespace"] },
  { resource: "hosts", module: "infra", permission: "infra:hosts:list", labelKey: "nav.hosts", icon: Server, group: "nav.infra", scopes: ["platform", "workspace", "namespace"] },
  { resource: "environments", module: "infra", permission: "infra:environments:list", labelKey: "nav.environments", icon: Layers, group: "nav.infra", scopes: ["platform", "workspace", "namespace"] },
  { resource: "regions", module: "infra", permission: "infra:regions:list", labelKey: "nav.regions", icon: MapPin, group: "nav.infra", scopes: ["platform"] },
  { resource: "sites", module: "infra", permission: "infra:sites:list", labelKey: "nav.sites", icon: Building2, group: "nav.infra", scopes: ["platform"] },
  { resource: "locations", module: "infra", permission: "infra:locations:list", labelKey: "nav.locations", icon: Warehouse, group: "nav.infra", scopes: ["platform"] },
  { resource: "logs", module: "audit", permission: "audit:logs:list", labelKey: "nav.auditLogs", icon: ScrollText, group: "nav.audit", scopes: ["platform"] },
]

// --- Derived maps ---

/** resource name → module prefix (e.g. "hosts" → "infra") */
const RESOURCE_MODULE_MAP: Record<string, string> = Object.fromEntries(
  NAV_ITEMS.map((item) => [item.resource, item.module]),
)

/** resource name → list permission code (e.g. "hosts" → "infra:hosts:list") */
const RESOURCE_PERMISSION_MAP: Record<string, string> = Object.fromEntries(
  NAV_ITEMS.map((item) => [item.resource, item.permission]),
)

/** All known resource names, derived from NAV_ITEMS. */
export const KNOWN_RESOURCES: string[] = Object.keys(RESOURCE_PERMISSION_MAP)

/** resource name → i18n label key (e.g. "hosts" → "nav.hosts"), used by breadcrumbs. */
export const RESOURCE_LABEL_KEYS: Record<string, string> = Object.fromEntries(
  NAV_ITEMS.map((item) => [item.resource, item.labelKey]),
)

// --- Navigation helpers ---

/** Look up the list permission code for a URL resource segment (e.g. "hosts" → "infra:hosts:list"). */
export function getResourcePermission(resource: string): string | undefined {
  return RESOURCE_PERMISSION_MAP[resource]
}

/** Extract the resource name the user is currently viewing from the URL path. */
export function detectResource(pathname: string): string | null {
  const segments = pathname.split("/").filter(Boolean)
  for (let i = segments.length - 1; i >= 0; i--) {
    if (KNOWN_RESOURCES.includes(segments[i])) return segments[i]
  }
  return null
}

/** Build a scope-aware URL path for a given resource. Derives module from NAV_ITEMS. */
export function buildScopedPath(
  resource: string | null,
  wsId: string | null,
  nsId: string | null,
): string {
  if (!resource) {
    if (wsId && nsId) return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
    if (wsId) return `/dashboard/workspaces/${wsId}/overview`
    return "/dashboard/overview"
  }
  const module = RESOURCE_MODULE_MAP[resource]
  if (!module) {
    if (wsId && nsId) return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
    if (wsId) return `/dashboard/workspaces/${wsId}/overview`
    return "/dashboard/overview"
  }
  if (wsId && nsId) return `/${module}/workspaces/${wsId}/namespaces/${nsId}/${resource}`
  if (wsId) return `/${module}/workspaces/${wsId}/${resource}`
  return `/${module}/${resource}`
}

/** Build permission scope object from URL params. */
export function buildPermScope(
  wsId?: string,
  nsId?: string,
): { workspaceId: string; namespaceId?: string } | undefined {
  if (nsId) return { workspaceId: wsId!, namespaceId: nsId }
  if (wsId) return { workspaceId: wsId }
  return undefined
}

/**
 * Dispatch an API call to the correct scope-level function based on wsId/nsId.
 * Eliminates the repetitive if/else if/else pattern across infra pages.
 */
export async function scopedApiCall<T>(
  wsId: string | undefined,
  nsId: string | undefined,
  platformFn: () => Promise<T>,
  workspaceFn: (wsId: string) => Promise<T>,
  namespaceFn: (wsId: string, nsId: string) => Promise<T>,
): Promise<T> {
  if (wsId && nsId) return namespaceFn(wsId, nsId)
  if (wsId) return workspaceFn(wsId)
  return platformFn()
}
