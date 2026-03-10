import { useCallback, useEffect, useState } from "react"
import { useParams, Link, Navigate } from "react-router"
import { Plus, Pencil, Trash2, Search } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Checkbox } from "@/components/ui/checkbox"
import {
  listNamespaceRoles, deleteNamespaceRole, deleteNamespaceRoles, listPermissions,
} from "@/api/iam/rbac"
import type { Role, Permission, ListParams } from "@/api/types"
import { ApiError, translateApiError } from "@/api/client"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { usePermission } from "@/hooks/use-permission"
import { usePermissionStore } from "@/stores/permission-store"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { ScopedRoleFormDialog } from "@/components/scoped-role-form-dialog"


export default function NamespaceRolesTab() {
  const workspaceId = useParams().workspaceId!
  const namespaceId = useParams().namespaceId!
  const { t } = useTranslation()
  const rolesBasePath = `/iam/workspaces/${workspaceId}/namespaces/${namespaceId}/roles`
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const permissionsLoaded = usePermissionStore((s) => s.permissions) !== null

  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [createOpen, setCreateOpen] = useState(false)
  const [editRole, setEditRole] = useState<Role | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchRoles = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      const data = await listNamespaceRoles(workspaceId, namespaceId, params)
      setRoles(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespaceId, page, pageSize, sortBy, sortOrder, search])

  useEffect(() => { fetchRoles() }, [fetchRoles])
  useEffect(() => { setPage(1) }, [search, pageSize])
  useEffect(() => { clearSelection() }, [roles])

  if (permissionsLoaded && !hasPermission("iam:namespaces:roles:list", { workspaceId, namespaceId })) {
    return <Navigate to="/" replace />
  }

  const canCreate = hasPermission("iam:namespaces:roles:create", { workspaceId, namespaceId })
  const canUpdate = hasPermission("iam:namespaces:roles:update", { workspaceId, namespaceId })
  const canDelete = hasPermission("iam:namespaces:roles:delete", { workspaceId, namespaceId })

  const loadPermissions = useCallback(async () => {
    if (permissions.length > 0) return
    try {
      const data = await listPermissions({ pageSize: 1000 })
      setPermissions(data.items ?? [])
    } catch {
      // silently ignore — permission selector will show empty
    }
  }, [permissions.length])

  const handleCreate = async () => {
    await loadPermissions()
    setCreateOpen(true)
  }

  const handleEdit = async (role: Role) => {
    await loadPermissions()
    setEditRole(role)
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteNamespaceRole(workspaceId, namespaceId, deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchRoles()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await deleteNamespaceRoles(workspaceId, namespaceId, Array.from(selected))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchRoles()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const selectableRoles = roles.filter((r) => !r.spec.builtin)
  const selectableIds = selectableRoles.map((r) => r.metadata.id)

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("role.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("role.manage", { count: totalCount })}</p>
        </div>
        {canCreate && (
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            {t("role.create")}
          </Button>
        )}
      </div>
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("role.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {canDelete && selected.size > 0 && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("role.batchDelete")} ({selected.size})
          </Button>
        )}
      </div>

      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              {canDelete && (
                <TableHead className="w-10">
                  <Checkbox
                    checked={selectableIds.length > 0 && selected.size === selectableIds.length}
                    onCheckedChange={() => toggleAll(selectableIds)}
                  />
                </TableHead>
              )}
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("role.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("role.builtin")}</TableHead>
              <TableHead>{t("common.description")}</TableHead>
              <TableHead>{t("role.rules")}</TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              {(canUpdate || canDelete) && <TableHead className="w-24">{t("common.actions")}</TableHead>}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>{Array.from({ length: 6 + (canDelete ? 1 : 0) + (canUpdate || canDelete ? 1 : 0) }).map((_, j) => (<TableCell key={j}><Skeleton className="h-4 w-20" /></TableCell>))}</TableRow>
              ))
            ) : roles.length === 0 ? (
              <TableRow><TableCell colSpan={6 + (canDelete ? 1 : 0) + (canUpdate || canDelete ? 1 : 0)} className="text-muted-foreground py-8 text-center">{t("role.noData")}</TableCell></TableRow>
            ) : (
              roles.map((role) => (
                <TableRow key={role.metadata.id}>
                  {canDelete && (
                    <TableCell>
                      <Checkbox
                        checked={selected.has(role.metadata.id)}
                        onCheckedChange={() => toggleOne(role.metadata.id)}
                        disabled={!!role.spec.builtin}
                      />
                    </TableCell>
                  )}
                  <TableCell className="font-medium">
                    <Link to={`${rolesBasePath}/${role.metadata.id}`} className="hover:underline">
                      {role.spec.name}
                    </Link>
                  </TableCell>
                  <TableCell>{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || "-" })}</TableCell>
                  <TableCell>
                    <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
                      {role.spec.builtin ? t("role.builtin") : t("role.custom")}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-48 truncate text-sm">
                    {t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "-" })}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {t("role.rulesCount", { count: role.spec.ruleCount ?? role.spec.rules?.length ?? 0 })}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(role.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  {(canUpdate || canDelete) && (
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {canUpdate && (
                          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleEdit(role)} disabled={!!role.spec.builtin} title={role.spec.builtin ? t("role.builtinCannotEdit") : t("common.edit")}>
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                        )}
                        {canDelete && (
                          <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(role)} disabled={!!role.spec.builtin} title={role.spec.builtin ? t("role.builtinCannotDelete") : t("common.delete")}>
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  )}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      <ScopedRoleFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        scope="namespace"
        scopeId={namespaceId}
        workspaceId={workspaceId}
        permissions={permissions}
        onSuccess={fetchRoles}
      />

      <ScopedRoleFormDialog
        open={!!editRole}
        onOpenChange={(v) => { if (!v) setEditRole(null) }}
        scope="namespace"
        scopeId={namespaceId}
        workspaceId={workspaceId}
        role={editRole ?? undefined}
        permissions={permissions}
        onSuccess={fetchRoles}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("role.deleteConfirm", { name: deleteTarget?.spec.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("role.batchDelete")}
        description={t("role.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}
