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
import { listWorkspaces } from "@/api/iam/workspaces"
import { listWorkspaceNamespaces } from "@/api/iam/namespaces"
import { useTranslation } from "@/i18n"
import type { Workspace, Namespace } from "@/api/types"

const ALL = "__all__"
const KNOWN_RESOURCES = ["overview", "users", "roles", "rolebindings", "namespaces", "workspaces"]

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
    if (resource === "overview") return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
    if (resource === "users" || resource === "roles" || resource === "rolebindings") return `${iamPrefix}/${resource}`
    return `/dashboard/workspaces/${wsId}/namespaces/${nsId}/overview`
  }
  if (wsId) {
    const iamPrefix = `/iam/workspaces/${wsId}`
    if (resource === "overview") return `/dashboard/workspaces/${wsId}/overview`
    if (resource === "users" || resource === "roles" || resource === "rolebindings" || resource === "namespaces")
      return `${iamPrefix}/${resource}`
    return `/dashboard/workspaces/${wsId}/overview`
  }
  // 平台范围
  if (resource === "overview") return "/dashboard/overview"
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

  useEffect(() => {
    if (!workspaceId) {
      setNamespaces([])
      return
    }
    let cancelled = false
    listWorkspaceNamespaces(workspaceId, { pageSize: 100 })
      .then((data) => {
        if (!cancelled) setNamespaces(data.items ?? [])
      })
      .catch(() => {
        if (!cancelled) setNamespaces([])
      })
    return () => {
      cancelled = true
    }
  }, [workspaceId])

  return (
    <div className="space-y-1">
      <Select
        value={workspaceId ?? ALL}
        onValueChange={(v) => {
          const wsId = v === ALL ? null : v
          setWorkspace(wsId)
          const resource = detectResource(location.pathname)
          navigate(buildScopedPath(resource, wsId, null))
        }}
      >
        <SelectTrigger
          size="sm"
          className="h-8 w-full gap-1.5 rounded-none border-0 bg-transparent px-3 text-xs shadow-none"
        >
          <Building2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>{t("scope.allWorkspaces")}</SelectItem>
          {workspaces.map((ws) => (
            <SelectItem key={ws.metadata.id} value={ws.metadata.id}>
              {ws.spec.displayName || ws.metadata.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={namespaceId ?? ALL}
        onValueChange={(v) => {
          const nsId = v === ALL ? null : v
          setNamespace(nsId)
          const resource = detectResource(location.pathname)
          navigate(buildScopedPath(resource, workspaceId, nsId))
        }}
        disabled={!workspaceId}
      >
        <SelectTrigger
          size="sm"
          className="h-8 w-full gap-1.5 rounded-none border-0 bg-transparent px-3 text-xs shadow-none"
        >
          <FolderKanban className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>{t("scope.allNamespaces")}</SelectItem>
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
