import { useCallback } from "react"
import { usePermissionStore } from "@/stores/permission-store"
import type { UserPermissionsSpec } from "@/api/types"

/**
 * Check if user has a specific permission at the given scope, using raw perms data.
 * This is a static version of hasPermission for use outside of React components.
 */
export function checkPermission(
  perms: UserPermissionsSpec,
  code: string,
  scope?: { workspaceId?: string; namespaceId?: string },
): boolean {
  if (perms.isPlatformAdmin) return true
  if (perms.platform?.includes(code)) return true

  if (scope?.namespaceId) {
    const nsPerms = perms.namespaces?.[scope.namespaceId]
    const wsId = scope.workspaceId || nsPerms?.workspaceId
    if (wsId) {
      const wsPerms = perms.workspaces?.[wsId]
      if (wsPerms?.permissions?.includes(code)) return true
    }
    if (nsPerms?.permissions?.includes(code)) return true
    return false
  }

  if (scope?.workspaceId) {
    const wsPerms = perms.workspaces?.[scope.workspaceId]
    if (wsPerms?.permissions?.includes(code)) return true
    return false
  }

  return false
}

/** Nav items in priority order for each scope level. */
const NAV_ITEMS: { path: string; permission: string; scope: "platform" | "workspace" | "namespace" }[] = [
  { path: "/dashboard/{scope}/overview", permission: "dashboard:overview:list", scope: "platform" },
  { path: "/iam/{scope}/workspaces", permission: "iam:workspaces:list", scope: "platform" },
  { path: "/iam/{scope}/namespaces", permission: "iam:namespaces:list", scope: "platform" },
  { path: "/iam/{scope}/users", permission: "iam:users:list", scope: "platform" },
  { path: "/iam/{scope}/roles", permission: "iam:roles:list", scope: "platform" },
  { path: "/iam/{scope}/rolebindings", permission: "iam:rolebindings:list", scope: "platform" },
  { path: "/infra/{scope}/hosts", permission: "infra:hosts:list", scope: "platform" },
  { path: "/infra/{scope}/environments", permission: "infra:environments:list", scope: "platform" },
  { path: "/audit/{scope}/logs", permission: "audit:logs:list", scope: "platform" },
]

function buildPath(template: string, wsId?: string, nsId?: string): string {
  if (wsId && nsId) return template.replace("{scope}/", `workspaces/${wsId}/namespaces/${nsId}/`)
  if (wsId) return template.replace("{scope}/", `workspaces/${wsId}/`)
  return template.replace("{scope}/", "")
}

/**
 * Compute the best landing page based on the user's permissions.
 * Finds the first nav item the user can access, checking platform → workspace → namespace scope.
 */
export function getDefaultPath(perms: UserPermissionsSpec): string {
  // Platform scope
  if (perms.isPlatformAdmin || (perms.platform?.length ?? 0) > 0) {
    for (const item of NAV_ITEMS) {
      if (checkPermission(perms, item.permission)) {
        return buildPath(item.path)
      }
    }
  }

  // Workspace scope
  const wsIds = Object.keys(perms.workspaces ?? {})
  if (wsIds.length > 0) {
    const wsId = wsIds[0]
    const wsScope = { workspaceId: wsId }
    for (const item of NAV_ITEMS) {
      if (checkPermission(perms, item.permission, wsScope)) {
        return buildPath(item.path, wsId)
      }
    }
  }

  // Namespace scope
  const nsEntries = Object.entries(perms.namespaces ?? {})
  if (nsEntries.length > 0) {
    const [nsId, nsPerms] = nsEntries[0]
    const nsScope = { workspaceId: nsPerms.workspaceId, namespaceId: nsId }
    for (const item of NAV_ITEMS) {
      if (checkPermission(perms, item.permission, nsScope)) {
        return buildPath(item.path, nsPerms.workspaceId, nsId)
      }
    }
  }

  return "/error?status=403"
}

/** Derived from NAV_ITEMS: resource name (last URL segment) → list permission code. */
const RESOURCE_PERMISSION_MAP: Record<string, string> = Object.fromEntries(
  NAV_ITEMS.map((item) => {
    const resource = item.path.split("/").pop()!
    return [resource, item.permission]
  }),
)

/** All known resource names, derived from NAV_ITEMS. */
export const KNOWN_RESOURCES: string[] = Object.keys(RESOURCE_PERMISSION_MAP)

/** Look up the list permission code for a URL resource segment (e.g. "hosts" → "infra:hosts:list"). */
export function getResourcePermission(resource: string): string | undefined {
  return RESOURCE_PERMISSION_MAP[resource]
}

/**
 * Find the first permitted navigation path for a specific scope.
 * Used by scope-selector to redirect when the current resource is not accessible in the new scope.
 */
export function getFirstPermittedPath(
  perms: UserPermissionsSpec,
  wsId: string | null,
  nsId: string | null,
): string {
  const scope = nsId && wsId
    ? { workspaceId: wsId, namespaceId: nsId }
    : wsId
      ? { workspaceId: wsId }
      : undefined
  for (const item of NAV_ITEMS) {
    if (checkPermission(perms, item.permission, scope)) {
      return buildPath(item.path, wsId ?? undefined, nsId ?? undefined)
    }
  }
  return "/error?status=403"
}

export function usePermission() {
  const permissions = usePermissionStore((s) => s.permissions)

  const isPlatformAdmin = permissions?.isPlatformAdmin ?? false

  const hasPermission = useCallback(
    (code: string, scope?: { workspaceId?: string; namespaceId?: string }): boolean => {
      if (!permissions) return false
      if (permissions.isPlatformAdmin) return true
      if (permissions.platform?.includes(code)) return true

      if (scope?.namespaceId) {
        const nsPerms = permissions.namespaces?.[scope.namespaceId]
        // Workspace perms cascade to namespace scope (inherit from explicit wsId or namespace's parent)
        const wsId = scope.workspaceId || nsPerms?.workspaceId
        if (wsId) {
          const wsPerms = permissions.workspaces?.[wsId]
          if (wsPerms?.permissions?.includes(code)) return true
        }
        // Namespace-specific perms
        if (nsPerms?.permissions?.includes(code)) return true
        return false
      }

      if (scope?.workspaceId) {
        const wsPerms = permissions.workspaces?.[scope.workspaceId]
        if (wsPerms?.permissions?.includes(code)) return true
        return false
      }

      return false
    },
    [permissions],
  )

  const hasAnyPermission = useCallback(
    (code: string): boolean => {
      if (!permissions) return false
      if (permissions.isPlatformAdmin) return true
      if (permissions.platform?.includes(code)) return true
      for (const ws of Object.values(permissions.workspaces ?? {})) {
        if (ws.permissions?.includes(code)) return true
      }
      for (const ns of Object.values(permissions.namespaces ?? {})) {
        if (ns.permissions?.includes(code)) return true
      }
      return false
    },
    [permissions],
  )

  return { hasPermission, hasAnyPermission, isPlatformAdmin }
}
