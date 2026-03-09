import { useCallback } from "react"
import { usePermissionStore } from "@/stores/permission-store"

export function usePermission() {
  const permissions = usePermissionStore((s) => s.permissions)

  const isPlatformAdmin = permissions?.isPlatformAdmin ?? false

  const hasPermission = useCallback(
    (code: string, scope?: { workspaceId?: string; namespaceId?: string }): boolean => {
      if (!permissions) return false
      if (permissions.isPlatformAdmin) return true
      if (permissions.platform.includes(code)) return true

      if (scope?.namespaceId) {
        const nsPerms = permissions.namespaces[scope.namespaceId]
        if (nsPerms) {
          // Check parent workspace permissions
          const wsPerms = permissions.workspaces[nsPerms.workspaceId]
          if (wsPerms?.permissions.includes(code)) return true
          // Check namespace own permissions
          if (nsPerms.permissions.includes(code)) return true
        }
        return false
      }

      if (scope?.workspaceId) {
        const wsPerms = permissions.workspaces[scope.workspaceId]
        if (wsPerms?.permissions.includes(code)) return true
        return false
      }

      // No scope → only platform-level (already checked above)
      return false
    },
    [permissions],
  )

  const hasAnyPermission = useCallback(
    (code: string): boolean => {
      if (!permissions) return false
      if (permissions.isPlatformAdmin) return true
      if (permissions.platform.includes(code)) return true
      for (const ws of Object.values(permissions.workspaces)) {
        if (ws.permissions.includes(code)) return true
      }
      for (const ns of Object.values(permissions.namespaces)) {
        if (ns.permissions.includes(code)) return true
      }
      return false
    },
    [permissions],
  )

  return { hasPermission, hasAnyPermission, isPlatformAdmin }
}
