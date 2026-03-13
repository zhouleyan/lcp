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
import { checkPermission, getFirstPermittedPath } from "@/hooks/use-permission"
import { detectResource, buildScopedPath, getResourcePermission, getScopeLevel, isResourceAtScope } from "@/lib/nav-config"
import { listWorkspaces } from "@/api/iam/workspaces"
import { listNamespaces, listWorkspaceNamespaces } from "@/api/iam/namespaces"
import { useTranslation } from "@/i18n"
import type { Workspace, Namespace } from "@/api/types"

const ALL = "__all__"
/** Re-fetch scope data every 5 minutes to detect membership changes made by others. */
const POLL_INTERVAL_MS = 5 * 60 * 1000

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

  // Fallback: fetch accessible namespaces via platform-level list API.
  // The backend injects AccessFilter for non-admin users which includes:
  // 1. Namespaces with direct namespace-scoped role bindings
  // 2. All namespaces in workspaces where the user has a workspace role with permission rules
  // This covers workspace-role users who don't have iam:namespaces:list specifically.
  const fetchAccessibleNamespaces = useCallback(async () => {
    if (!workspaceId) {
      setNamespaces([])
      return
    }
    try {
      const data = await listNamespaces({ pageSize: 100 })
      const filtered = (data.items ?? []).filter((ns) => ns.spec.workspaceId === workspaceId)
      setNamespaces(filtered)
    } catch {
      // Keep previous data on error to avoid flickering
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId])

  // Workspace-role users (with iam:namespaces:list) see all namespaces via standard API.
  // Namespace-only members fall back to their joined namespaces.
  const canListNamespaces = permissions
    ? checkPermission(permissions, "iam:namespaces:list", workspaceId ? { workspaceId } : undefined)
    : false

  useEffect(() => {
    if (canListNamespaces) {
      fetchNamespaces()
    } else {
      fetchAccessibleNamespaces()
    }
  }, [fetchNamespaces, fetchAccessibleNamespaces, namespaceId, version, canListNamespaces])

  // Stale namespace detection + auto-select for non-platform users
  useEffect(() => {
    if (namespaceId && namespaces.length > 0 && !namespaces.some((ns) => ns.metadata.id === namespaceId)) {
      // Current namespace no longer accessible — redirect
      const newNsId = namespaces[0]?.metadata.id ?? null
      if (permissions) {
        navigate(getFirstPermittedPath(permissions, workspaceId, newNsId))
      } else {
        const resource = detectResource(location.pathname)
        navigate(buildScopedPath(resource, workspaceId, newNsId))
      }
    } else if (!hasPlatformScope && !hasWorkspaceScope && namespaces.length === 1 && !namespaceId) {
      // Non-platform user with single namespace — auto-select
      const nsId = namespaces[0].metadata.id
      const resource = detectResource(location.pathname)
      navigate(buildScopedPath(resource, workspaceId, nsId))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasPlatformScope, hasWorkspaceScope, namespaces, namespaceId])

  return (
    <div className="space-y-1">
      <Select
        value={workspaceId ?? (hasPlatformScope ? ALL : "")}
        onValueChange={(v) => {
          const wsId = v === ALL ? null : v
          const resource = detectResource(location.pathname)
          const targetScope = getScopeLevel(wsId, null)
          const available = resource ? isResourceAtScope(resource, targetScope) : false
          const permCode = available && resource ? getResourcePermission(resource) : undefined
          const scope = wsId ? { workspaceId: wsId } : undefined
          // Navigate only; root-layout's useLayoutEffect syncs scope store from URL,
          // avoiding stale requests from the old page seeing the new scope before unmounting.
          if (available && permCode && permissions && checkPermission(permissions, permCode, scope)) {
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
          const targetScope = getScopeLevel(workspaceId, nsId)
          const available = resource ? isResourceAtScope(resource, targetScope) : false
          const permCode = available && resource ? getResourcePermission(resource) : undefined
          const scope = nsId && workspaceId
            ? { workspaceId, namespaceId: nsId }
            : workspaceId ? { workspaceId } : undefined
          if (available && permCode && permissions && checkPermission(permissions, permCode, scope)) {
            navigate(buildScopedPath(resource, workspaceId, nsId))
          } else if (permissions) {
            navigate(getFirstPermittedPath(permissions, workspaceId, nsId))
          } else {
            navigate(buildScopedPath(resource, workspaceId, nsId))
          }
        }}
        onOpenChange={(open) => { if (open) { canListNamespaces ? fetchNamespaces() : fetchAccessibleNamespaces() } }}
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
          {namespaces.map((ns) => (
            <SelectItem key={ns.metadata.id} value={ns.metadata.id}>
              {ns.spec.displayName || ns.metadata.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
