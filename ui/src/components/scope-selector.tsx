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
import { listWorkspaces } from "@/api/iam/workspaces"
import { listNamespaces, listWorkspaceNamespaces } from "@/api/iam/namespaces"
import { useTranslation } from "@/i18n"
import type { Workspace, Namespace } from "@/api/types"

const ALL = "__all__"
const KNOWN_RESOURCES = ["overview", "users", "roles", "rolebindings", "namespaces", "workspaces", "hosts", "environments"]

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
  const setWorkspace = useScopeStore((s) => s.setWorkspace)
  const setNamespace = useScopeStore((s) => s.setNamespace)

  const permissions = usePermissionStore((s) => s.permissions)
  const hasPlatformScope = permissions?.isPlatformAdmin || (permissions?.platform?.length ?? 0) > 0
  // Workspace-level users (e.g. workspace-viewer) should see "All namespaces" within their workspace
  const hasWorkspaceScope = !!(workspaceId && (permissions?.workspaces?.[workspaceId]?.permissions?.length ?? 0) > 0)

  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [namespaces, setNamespaces] = useState<Namespace[]>([])

  const fetchWorkspaces = useCallback(async () => {
    try {
      const data = await listWorkspaces({ pageSize: 100 })
      setWorkspaces(data.items ?? [])
    } catch {
      setWorkspaces([])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    fetchWorkspaces()
  }, [fetchWorkspaces])

  // Non-platform users: auto-select the only workspace
  useEffect(() => {
    if (!hasPlatformScope && workspaces.length > 0 && !workspaceId) {
      const firstWsId = workspaces[0].metadata.id
      setWorkspace(firstWsId)
      const resource = detectResource(location.pathname)
      navigate(buildScopedPath(resource, firstWsId, null))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasPlatformScope, workspaces, workspaceId])

  // Fetch namespaces scoped to the selected workspace (avoids fetching all namespaces).
  // Falls back to platform-level list when no workspace is selected (for non-platform users during init).
  const fetchNamespaces = useCallback(async () => {
    try {
      const data = workspaceId
        ? await listWorkspaceNamespaces(workspaceId, { pageSize: 100 })
        : await listNamespaces({ pageSize: 100 })
      setNamespaces(data.items ?? [])
    } catch {
      setNamespaces([])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId])

  useEffect(() => {
    fetchNamespaces()
  }, [fetchNamespaces])

  const accessibleNamespaces = namespaces

  // Non-platform users without workspace scope and with single namespace: auto-select
  useEffect(() => {
    if (!hasPlatformScope && !hasWorkspaceScope && accessibleNamespaces.length === 1 && !namespaceId) {
      const nsId = accessibleNamespaces[0].metadata.id
      setNamespace(nsId)
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
          setWorkspace(wsId)
          const resource = detectResource(location.pathname)
          navigate(buildScopedPath(resource, wsId, null))
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
          setNamespace(nsId)
          const resource = detectResource(location.pathname)
          navigate(buildScopedPath(resource, workspaceId, nsId))
        }}
        onOpenChange={(open) => { if (open) fetchNamespaces() }}
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
