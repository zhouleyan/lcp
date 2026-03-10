import { useCallback } from "react"
import { usePermissionStore } from "@/stores/permission-store"
import type { UserPermissionsSpec } from "@/api/types"

/**
 * Compute the best landing page based on the user's permission scope.
 * Platform perms → platform overview; workspace-only → that workspace's overview; etc.
 */
export function getDefaultPath(perms: UserPermissionsSpec): string {
  if (perms.isPlatformAdmin || (perms.platform?.length ?? 0) > 0) {
    return "/dashboard/overview"
  }
  const wsIds = Object.keys(perms.workspaces ?? {})
  if (wsIds.length > 0) {
    return `/dashboard/workspaces/${wsIds[0]}/overview`
  }
  const nsEntries = Object.entries(perms.namespaces ?? {})
  if (nsEntries.length > 0) {
    const [nsId, nsPerms] = nsEntries[0]
    return `/dashboard/workspaces/${nsPerms.workspaceId}/namespaces/${nsId}/overview`
  }
  // Zero-permission users get redirected to 403 by RootLayout; avoid extra redirect hop.
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
