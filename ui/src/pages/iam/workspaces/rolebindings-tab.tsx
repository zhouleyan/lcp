import { useCallback, useEffect, useState } from "react"
import { useParams, Navigate } from "react-router"
import { Plus, Trash2, Search } from "lucide-react"
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
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  listWorkspaceRoleBindings, createWorkspaceRoleBinding, deleteWorkspaceRoleBinding, listWorkspaceRoles,
} from "@/api/iam/rbac"
import { listUsers } from "@/api/iam/users"
import type { RoleBinding, Role, User, ListParams } from "@/api/types"
import { ApiError, translateApiError } from "@/api/client"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { usePermissionStore } from "@/stores/permission-store"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"


export default function WorkspaceRoleBindingsTab() {
  const workspaceId = useParams().workspaceId!
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()

  const permissionsLoaded = usePermissionStore((s) => s.permissions) !== null

  const [bindings, setBindings] = useState<RoleBinding[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<RoleBinding | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchBindings = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      const data = await listWorkspaceRoleBindings(workspaceId, params)
      setBindings(data.items ?? [])
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
  }, [workspaceId, page, pageSize, sortBy, sortOrder, search])

  useEffect(() => { fetchBindings() }, [fetchBindings])
  useEffect(() => { setPage(1) }, [search, pageSize])
  useEffect(() => { clearSelection() }, [bindings])

  if (permissionsLoaded && !hasPermission("iam:workspaces:rolebindings:list", { workspaceId })) {
    return <Navigate to="/" replace />
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteWorkspaceRoleBinding(workspaceId, deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchBindings()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("rolebinding.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await Promise.all(Array.from(selected).map((id) => deleteWorkspaceRoleBinding(workspaceId, id)))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchBindings()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("rolebinding.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const selectableBindings = bindings.filter((b) => !b.spec.isOwner)
  const selectableIds = selectableBindings.map((b) => b.metadata.id)

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("rolebinding.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("rolebinding.manage", { count: totalCount })}</p>
        </div>
        {hasPermission("iam:workspaces:rolebindings:create", { workspaceId }) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("rolebinding.create")}
          </Button>
        )}
      </div>
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("rolebinding.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission("iam:workspaces:rolebindings:delete", { workspaceId }) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("rolebinding.batchDelete")} ({selected.size})
          </Button>
        )}
      </div>

      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {hasPermission("iam:workspaces:rolebindings:delete", { workspaceId }) && (
                  <Checkbox
                    checked={selectableIds.length > 0 && selected.size === selectableIds.length}
                    onCheckedChange={() => toggleAll(selectableIds)}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("username")}>
                {t("user.username")}<SortIcon field="username" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("user_display_name")}>
                {t("common.displayName")}<SortIcon field="user_display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("role_name")}>
                {t("rolebinding.role")}<SortIcon field="role_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("role_display_name")}>
                {t("rolebinding.roleDisplayName")}<SortIcon field="role_display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-16">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>{Array.from({ length: 7 }).map((_, j) => (<TableCell key={j}><Skeleton className="h-4 w-20" /></TableCell>))}</TableRow>
              ))
            ) : bindings.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="text-muted-foreground py-8 text-center">{t("rolebinding.noData")}</TableCell></TableRow>
            ) : (
              bindings.map((binding) => (
                <TableRow key={binding.metadata.id}>
                  <TableCell>
                    {hasPermission("iam:workspaces:rolebindings:delete", { workspaceId }) && (
                      <Checkbox
                        checked={selected.has(binding.metadata.id)}
                        onCheckedChange={() => toggleOne(binding.metadata.id)}
                        disabled={!!binding.spec.isOwner}
                      />
                    )}
                  </TableCell>
                  <TableCell className="font-medium">{binding.spec.username}</TableCell>
                  <TableCell>{binding.spec.userDisplayName || "-"}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {t(`role.${binding.spec.roleName}`, { defaultValue: binding.spec.roleDisplayName || binding.spec.roleName || "" })}
                    </Badge>
                  </TableCell>
                  <TableCell>{binding.spec.roleDisplayName || "-"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(binding.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    {hasPermission("iam:workspaces:rolebindings:delete", { workspaceId }) && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(binding)}
                        disabled={!!binding.spec.isOwner}
                        title={t("common.delete")}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      <CreateWorkspaceRoleBindingDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        workspaceId={workspaceId}
        onSuccess={fetchBindings}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("rolebinding.deleteConfirm", { name: deleteTarget?.spec.username ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("rolebinding.batchDelete")}
        description={t("rolebinding.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Create Workspace RoleBinding Dialog =====

function CreateWorkspaceRoleBindingDialog({
  open, onOpenChange, workspaceId, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspaceId: string
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [roles, setRoles] = useState<Role[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loadingRoles, setLoadingRoles] = useState(false)
  const [loadingUsers, setLoadingUsers] = useState(false)
  const [selectedRoleId, setSelectedRoleId] = useState("")
  const [selectedUserId, setSelectedUserId] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")

  useEffect(() => {
    if (open) {
      setSelectedRoleId("")
      setSelectedUserId("")
      setSearchQuery("")
      setLoadingRoles(true)
      setLoadingUsers(true)
      listWorkspaceRoles(workspaceId, { pageSize: 100 })
        .then((data) => setRoles(data.items ?? []))
        .catch(() => {})
        .finally(() => setLoadingRoles(false))
      listUsers({ pageSize: 100 })
        .then((data) => setUsers(data.items ?? []))
        .catch(() => {})
        .finally(() => setLoadingUsers(false))
    }
  }, [open, workspaceId])

  const filteredUsers = searchQuery
    ? users.filter((u) => {
        const q = searchQuery.toLowerCase()
        return (
          u.spec.username.toLowerCase().includes(q) ||
          (u.spec.email?.toLowerCase().includes(q)) ||
          (u.spec.displayName?.toLowerCase().includes(q)) ||
          (u.spec.phone?.includes(q))
        )
      })
    : users

  const handleSubmit = async () => {
    if (!selectedUserId || !selectedRoleId) return
    setSubmitting(true)
    try {
      await createWorkspaceRoleBinding(workspaceId, {
        spec: { userId: selectedUserId, roleId: selectedRoleId, scope: "workspace" },
      })
      toast.success(t("action.createSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("rolebinding.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{t("rolebinding.create")}</DialogTitle>
          <DialogDescription>{t("rolebinding.createDesc")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <p className="mb-2 text-sm font-medium">{t("rolebinding.selectRole")}</p>
            <div className="max-h-[160px] overflow-auto border">
              {loadingRoles ? (
                <div className="space-y-2 p-4">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-8 w-full" />)}</div>
              ) : roles.length === 0 ? (
                <p className="text-muted-foreground p-4 text-center text-sm">{t("rolebinding.noRoles")}</p>
              ) : (
                roles.map((role) => (
                  <label
                    key={role.metadata.id}
                    className="flex cursor-pointer items-center gap-3 px-4 py-2 hover:bg-muted/50"
                  >
                    <Checkbox
                      checked={selectedRoleId === role.metadata.id}
                      onCheckedChange={() => setSelectedRoleId(selectedRoleId === role.metadata.id ? "" : role.metadata.id)}
                    />
                    <div className="flex-1">
                      <p className="text-sm font-medium">{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || role.spec.name })}</p>
                      <p className="text-muted-foreground text-xs">{t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "" }) || "-"}</p>
                    </div>
                  </label>
                ))
              )}
            </div>
          </div>

          <div>
            <p className="mb-2 text-sm font-medium">{t("rolebinding.selectUser")}</p>
            <div className="relative mb-2">
              <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
              <Input
                placeholder={t("common.search")}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            <div className="max-h-[200px] overflow-auto border">
              {loadingUsers ? (
                <div className="space-y-2 p-4">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-8 w-full" />)}</div>
              ) : filteredUsers.length === 0 ? (
                <p className="text-muted-foreground p-4 text-center text-sm">{searchQuery ? t("common.noSearchResults") : t("rolebinding.noUsers")}</p>
              ) : (
                filteredUsers.map((user) => (
                  <label
                    key={user.metadata.id}
                    className="flex cursor-pointer items-center gap-3 px-4 py-2 hover:bg-muted/50"
                  >
                    <Checkbox
                      checked={selectedUserId === user.metadata.id}
                      onCheckedChange={() => setSelectedUserId(selectedUserId === user.metadata.id ? "" : user.metadata.id)}
                    />
                    <div className="flex-1">
                      <p className="text-sm font-medium">{user.spec.username}</p>
                      <p className="text-muted-foreground text-xs">{user.spec.displayName || user.spec.email}</p>
                    </div>
                  </label>
                ))
              )}
            </div>
          </div>
        </div>

        <DialogFooter className="mt-6 pt-4 border-t">
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={!selectedUserId || !selectedRoleId || submitting}>
            {submitting ? "..." : t("rolebinding.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
