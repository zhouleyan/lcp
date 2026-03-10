import { useCallback, useEffect, useMemo, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Pencil, Trash2, ArrowLeft } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription,
} from "@/components/ui/dialog"
import {
  getWorkspaceRole, deleteWorkspaceRole,
  getNamespaceRole, deleteNamespaceRole,
  listPermissions,
} from "@/api/iam/rbac"
import { ApiError, translateApiError } from "@/api/client"
import type { Role, Permission } from "@/api/types"
import { useTranslation } from "@/i18n"
import { PermissionSelector, patternCovers } from "@/components/permission-selector"
import { ScopedRoleFormDialog } from "@/components/scoped-role-form-dialog"

export default function ScopedRoleDetailPage() {
  const { workspaceId, namespaceId, roleId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [role, setRole] = useState<Role | null>(null)
  const [loading, setLoading] = useState(true)
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const scope = namespaceId ? "namespace" : "workspace"
  const scopeId = namespaceId ?? workspaceId!

  // Build base path for back navigation
  const basePath = namespaceId
    ? `/iam/workspaces/${workspaceId}/namespaces/${namespaceId}/roles`
    : `/iam/workspaces/${workspaceId}/roles`

  const fetchRole = useCallback(async () => {
    if (!roleId) return
    try {
      const r = namespaceId
        ? await getNamespaceRole(workspaceId!, namespaceId, roleId)
        : await getWorkspaceRole(workspaceId!, roleId)
      setRole(r)
    } catch {
      setRole(null)
    } finally {
      setLoading(false)
    }
  }, [roleId, workspaceId, namespaceId])

  useEffect(() => { fetchRole() }, [fetchRole])

  useEffect(() => {
    listPermissions({ pageSize: 1000 })
      .then((data) => setPermissions(data.items ?? []))
      .catch(() => {})
  }, [])

  const handleDelete = async () => {
    if (!role) return
    try {
      if (namespaceId) {
        await deleteNamespaceRole(workspaceId!, namespaceId, role.metadata.id)
      } else {
        await deleteWorkspaceRole(workspaceId!, role.metadata.id)
      }
      toast.success(t("action.deleteSuccess"))
      navigate(basePath)
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!role) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("role.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate(basePath)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold">{role.spec.name}</h1>
          <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
            {role.spec.builtin ? t("role.builtin") : t("role.custom")}
          </Badge>
        </div>
        {!role.spec.builtin && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          </div>
        )}
      </div>

      {/* role info card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>{t("role.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("role.name")}</span>
              <p className="font-medium">{role.spec.name}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || "-" })}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("role.scope")}</span>
              <p>
                <Badge variant={scope === "workspace" ? "secondary" : "outline"}>
                  {t(`role.scope.${scope}`)}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("role.builtin")}</span>
              <p>
                <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
                  {role.spec.builtin ? t("role.builtin") : t("role.custom")}
                </Badge>
              </p>
            </div>
            <div className="col-span-2">
              <span className="text-muted-foreground">{t("common.description")}</span>
              <p className="font-medium">{t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "-" })}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(role.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(role.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* permission rules card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>
            {t("role.rules")}
            <span className="text-muted-foreground ml-2 text-sm font-normal">
              ({t("role.rulesCount", { count: role.spec.rules?.length ?? 0 })})
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {(!role.spec.rules || role.spec.rules.length === 0) ? (
            <p className="text-muted-foreground text-sm">{t("role.noPermissions")}</p>
          ) : (
            <div className="grid grid-cols-2 gap-6" style={{ height: "min(600px, 60vh)" }}>
              <div className="overflow-y-auto rounded-md border">
                <PermissionSelector
                  permissions={permissions}
                  value={role.spec.rules}
                  readOnly
                />
              </div>
              <div className="overflow-y-auto rounded-md border">
                <MatchedRulesList
                  rules={role.spec.rules}
                  permissions={permissions}
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* edit dialog */}
      {!role.spec.builtin && (
        <ScopedRoleFormDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          scope={scope}
          scopeId={scopeId}
          workspaceId={workspaceId}
          role={role}
          permissions={permissions}
          onSuccess={fetchRole}
        />
      )}

      {/* delete confirm */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.delete")}</DialogTitle>
            <DialogDescription>
              {t("role.deleteConfirm", { name: role.spec.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleDelete}>{t("common.delete")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ===== Matched Rules List =====

function MatchedRulesList({
  rules,
  permissions,
}: {
  rules: string[]
  permissions: Permission[]
}) {
  const { t } = useTranslation()

  const ruleDetails = useMemo(() => {
    return rules.map((rule) => {
      const matched = rule.includes("*")
        ? permissions.filter((p) => patternCovers(rule, p.spec.code))
        : permissions.filter((p) => p.spec.code === rule)
      return { rule, matched }
    })
  }, [rules, permissions])

  return (
    <div>
      {ruleDetails.map(({ rule, matched }) => (
        <div key={rule} className="border-b last:border-b-0">
          <div className="flex items-center justify-between bg-muted/30 px-3 py-1.5">
            <code className="text-xs font-medium">{rule}</code>
            <span className="text-muted-foreground text-xs">
              {t("role.matchCount", { count: matched.length })}
            </span>
          </div>
          {matched.length > 0 && (
            <div className="px-3 py-1">
              {matched.map((p) => (
                <p key={p.spec.code} className="text-muted-foreground truncate py-0.5 text-xs">
                  {t(`perm.${p.spec.code}`, { defaultValue: p.spec.description || p.spec.code })}
                </p>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
