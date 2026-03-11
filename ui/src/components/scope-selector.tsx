import { useEffect, useState, useCallback } from "react"
import { useNavigate, useLocation } from "react-router"
import { Building2, FolderKanban } from "lucide-react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useScopeStore } from "@/stores/scope-store"
import { usePermissionStore } from "@/stores/permission-store"
import { checkPermission, getResourcePermission, getFirstPermittedPath, KNOWN_RESOURCES } from "@/hooks/use-permission"
import { listWorkspaces } from "@/api/iam/workspaces"
import { listNamespaces, listWorkspaceNamespaces } from "@/api/iam/namespaces"
import { listUserNamespaces } from "@/api/iam/users"
import { useAuthStore } from "@/stores/auth-store"
import { useTranslation } from "@/i18n"
import type { Workspace, Namespace } from "@/api/types"

const ALL = "__all__"
/** Re-fetch scope data every 5 minutes to detect membership changes made by others. */
const POLL_INTERVAL_MS = 5 * 60 * 1000

/** 从当前路径中提取用户正在查看的资源类型 */
function detectResource(pathname: string): string | null {
  const segments = pathname.split("/").filter(Boolean)
  for (let i = segments.length - 1; i >= 0; i--) {
    if (KNOWN_RESOURCES.includes(segments[i])) return segments[i]
  }
  return null
}

/** 根据目标 scope 和当前资源类型，构建导航路径 */
function buildScopedPath(
  resource: string | null,
  wsId: string | null,
  nsId: string | null,
): string {
  if (wsId && nsId) {
    const iamPrefix = `/iam/workspaces/${wsId}/namespaces/${nsId}`
    const infraPrefix = `/infra/workspaces/${wsId}/namespaces/${nsId}`
    if (resource === "overview") return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
    if (resource === "users" || resource === "roles" || resource === "rolebindings") return `${iamPrefix}/${resource}`
    if (resource === "hosts" || resource === "environments") return `${infraPrefix}/${resource}`
    return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
  }
  if (wsId) {
    const iamPrefix = `/iam/workspaces/${wsId}`
    const infraPrefix = `/infra/workspaces/${wsId}`
    if (resource === "overview") return `/dashboard/workspaces/${wsId}/overview`
    if (resource === "users" || resource === "roles" || resource === "rolebindings" || resource === "namespaces")
      return `${iamPrefix}/${resource}`
    if (resource === "hosts" || resource === "environments") return `${infraPrefix}/${resource}`
    return `/dashboard/workspaces/${wsId}/overview`
  }
  // 平台范围
  if (resource === "overview") return "/dashboard/overview"
  if (resource === "hosts" || resource === "environments") return `/infra/${resource}`
  if (resource && KNOWN_RESOURCES.includes(resource)) return `/iam/${resource}`
  return "/dashboard/overview"
}

export function ScopeSelector() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const workspaceId = useScopeStore((s) => s.workspaceId)
  const namespaceId = useScopeStore((s) => s.namespaceId)
  const version = useScopeStore((s) => s.version)
  const invalidate = useScopeStore((s) => s.invalidate)

  const permissions = usePermissionStore((s) => s.permissions)
  const hasPlatformScope = permissions?.isPlatformAdmin || (permissions?.platform?.length ?? 0) > 0
  // Workspace-level users (e.g. workspace-viewer) should see "All namespaces" within their workspace
  const hasWorkspaceScope = !!(workspaceId && (permissions?.workspaces?.[workspaceId]?.permissions?.length ?? 0) > 0)

  const userId = useAuthStore((s) => s.user?.sub)

  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [namespaces, setNamespaces] = useState<Namespace[]>([])

  // Periodic polling: bump version to re-fetch scope data
  useEffect(() => {
    const timer = setInterval(() => invalidate(), POLL_INTERVAL_MS)
    return () => clearInterval(timer)
  }, [invalidate])

  const fetchWorkspaces = useCallback(async () => {
    try {
      const data = await listWorkspaces({ pageSize: 100 })
      setWorkspaces(data.items ?? [])
    } catch {
      // Keep previous data on error to avoid flickering
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    fetchWorkspaces()
  }, [fetchWorkspaces, workspaceId, version])

  // Stale workspace detection: if current workspace was removed, redirect to first permitted path
  // Non-platform users with no workspace selected: auto-select
  useEffect(() => {
    if (workspaces.length === 0) return
    if (workspaceId && !workspaces.some((ws) => ws.metadata.id === workspaceId)) {
      // Current workspace no longer accessible — redirect
      const newWsId = workspaces[0]?.metadata.id ?? null
      if (permissions) {
        navigate(getFirstPermittedPath(permissions, newWsId, null))
      } else {
        const resource = detectResource(location.pathname)
        navigate(buildScopedPath(resource, newWsId, null))
      }
    } else if (!hasPlatformScope && !workspaceId) {
      // Non-platform user with no workspace — auto-select first
      const firstWsId = workspaces[0].metadata.id
      const resource = detectResource(location.pathname)
      navigate(buildScopedPath(resource, firstWsId, null))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasPlatformScope, workspaces, workspaceId])

  // Fetch all namespaces via standard list API (requires iam:namespaces:list permission).
  const fetchNamespaces = useCallback(async () => {
    try {
      const data = workspaceId
        ? await listWorkspaceNamespaces(workspaceId, { pageSize: 100 })
        : await listNamespaces({ pageSize: 100 })
      setNamespaces(data.items ?? [])
    } catch {
      // Keep previous data on error to avoid flickering
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId])

  // Fallback: fetch user's joined namespaces when user lacks iam:namespaces:list.
  // Uses /users/{userId}:namespaces which doesn't require namespace list permission.
  const fetchUserJoinedNamespaces = useCallback(async () => {
    if (!userId || !workspaceId) {
      setNamespaces([])
      return
    }
    try {
      const data = await listUserNamespaces(userId, { pageSize: 100 })
      const filtered = (data.items ?? []).filter((ns) => ns.spec.workspaceId === workspaceId)
      setNamespaces(filtered)
    } catch {
      // Keep previous data on error to avoid flickering
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId, workspaceId])

  // Workspace-role users (with iam:namespaces:list) see all namespaces via standard API.
  // Namespace-only members fall back to their joined namespaces.
  const canListNamespaces = permissions
    ? checkPermission(permissions, "iam:namespaces:list", workspaceId ? { workspaceId } : undefined)
    : false

  useEffect(() => {
    if (canListNamespaces) {
      fetchNamespaces()
    } else {
      fetchUserJoinedNamespaces()
    }
  }, [fetchNamespaces, fetchUserJoinedNamespaces, namespaceId, version, canListNamespaces])

  const accessibleNamespaces = namespaces

  // Stale namespace detection + auto-select for non-platform users
  useEffect(() => {
    if (namespaceId && accessibleNamespaces.length > 0 && !accessibleNamespaces.some((ns) => ns.metadata.id === namespaceId)) {
      // Current namespace no longer accessible — redirect
      const newNsId = accessibleNamespaces[0]?.metadata.id ?? null
      if (permissions) {
        navigate(getFirstPermittedPath(permissions, workspaceId, newNsId))
      } else {
        const resource = detectResource(location.pathname)
        navigate(buildScopedPath(resource, workspaceId, newNsId))
      }
    } else if (!hasPlatformScope && !hasWorkspaceScope && accessibleNamespaces.length === 1 && !namespaceId) {
      // Non-platform user with single namespace — auto-select
      const nsId = accessibleNamespaces[0].metadata.id
      const resource = detectResource(location.pathname)
      navigate(buildScopedPath(resource, workspaceId, nsId))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasPlatformScope, hasWorkspaceScope, accessibleNamespaces, namespaceId])

  return (
    <div className="space-y-1">
      <Select
        value={workspaceId ?? (hasPlatformScope ? ALL : "")}
        onValueChange={(v) => {
          const wsId = v === ALL ? null : v
          const resource = detectResource(location.pathname)
          const permCode = resource ? getResourcePermission(resource) : undefined
          const scope = wsId ? { workspaceId: wsId } : undefined
          // Navigate only; root-layout's useLayoutEffect syncs scope store from URL,
          // avoiding stale requests from the old page seeing the new scope before unmounting.
          if (permCode && permissions && checkPermission(permissions, permCode, scope)) {
            navigate(buildScopedPath(resource, wsId, null))
          } else if (permissions) {
            navigate(getFirstPermittedPath(permissions, wsId, null))
          } else {
            navigate(buildScopedPath(resource, wsId, null))
          }
        }}
        onOpenChange={(open) => { if (open) fetchWorkspaces() }}
      >
        <SelectTrigger
          size="sm"
          className="h-8 w-full gap-1.5 rounded-none border-0 bg-transparent px-3 text-xs shadow-none"
        >
          <Building2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <SelectValue placeholder={t("scope.selectWorkspace")} />
        </SelectTrigger>
        <SelectContent>
          {hasPlatformScope && <SelectItem value={ALL}>{t("scope.allWorkspaces")}</SelectItem>}
          {workspaces.map((ws) => (
            <SelectItem key={ws.metadata.id} value={ws.metadata.id}>
              {ws.spec.displayName || ws.metadata.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={namespaceId ?? ((hasPlatformScope || hasWorkspaceScope) ? ALL : "")}
        onValueChange={(v) => {
          const nsId = v === ALL ? null : v
          const resource = detectResource(location.pathname)
          const permCode = resource ? getResourcePermission(resource) : undefined
          const scope = nsId && workspaceId
            ? { workspaceId, namespaceId: nsId }
            : workspaceId ? { workspaceId } : undefined
          if (permCode && permissions && checkPermission(permissions, permCode, scope)) {
            navigate(buildScopedPath(resource, workspaceId, nsId))
          } else if (permissions) {
            navigate(getFirstPermittedPath(permissions, workspaceId, nsId))
          } else {
            navigate(buildScopedPath(resource, workspaceId, nsId))
          }
        }}
        onOpenChange={(open) => { if (open) { canListNamespaces ? fetchNamespaces() : fetchUserJoinedNamespaces() } }}
        disabled={!workspaceId}
      >
        <SelectTrigger
          size="sm"
          className="h-8 w-full gap-1.5 rounded-none border-0 bg-transparent px-3 text-xs shadow-none"
        >
          <FolderKanban className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <SelectValue placeholder={t("scope.selectNamespace")} />
        </SelectTrigger>
        <SelectContent>
          {(hasPlatformScope || hasWorkspaceScope) && <SelectItem value={ALL}>{t("scope.allNamespaces")}</SelectItem>}
          {accessibleNamespaces.map((ns) => (
            <SelectItem key={ns.metadata.id} value={ns.metadata.id}>
              {ns.spec.displayName || ns.metadata.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
