import { useEffect, useState, useCallback } from "react"
import { Building2, FolderKanban } from "lucide-react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useScopeStore } from "@/stores/scope-store"
import { listWorkspaces } from "@/api/workspaces"
import { listWorkspaceNamespaces } from "@/api/namespaces"
import { useTranslation } from "@/i18n"
import type { Workspace, Namespace } from "@/api/types"

const ALL = "__all__"

export function ScopeSelector() {
  const { t } = useTranslation()
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
        onValueChange={(v) => setWorkspace(v === ALL ? null : v)}
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
        onValueChange={(v) => setNamespace(v === ALL ? null : v)}
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
