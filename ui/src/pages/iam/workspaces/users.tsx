import { useCallback, useEffect, useState } from "react"
import { useParams, Link } from "react-router"
import { Plus, UserMinus, Search, Filter } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { Checkbox } from "@/components/ui/checkbox"
import {
  listWorkspaceUsers, addWorkspaceUsers, removeWorkspaceUsers, listUsers,
} from "@/api/iam/users"
import { listWorkspaceRoles, createWorkspaceRoleBinding } from "@/api/iam/rbac"
import type { User, Role, ListParams } from "@/api/types"
import { ApiError, translateApiError } from "@/api/client"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"


export default function WorkspaceUsersPage() {
  const workspaceId = useParams().workspaceId!
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()

  const [members, setMembers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")

  const [addOpen, setAddOpen] = useState(false)
  const [removeTarget, setRemoveTarget] = useState<User | null>(null)
  const [batchRemoveOpen, setBatchRemoveOpen] = useState(false)

  const fetchMembers = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      const data = await listWorkspaceUsers(workspaceId, params)
      setMembers(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("user.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId, page, pageSize, sortBy, sortOrder, search, statusFilter])

  useEffect(() => { fetchMembers() }, [fetchMembers])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])
  useEffect(() => { clearSelection() }, [members])

  const handleRemove = async () => {
    if (!removeTarget) return
    try {
      await removeWorkspaceUsers(workspaceId, [removeTarget.metadata.id])
      toast.success(t("workspace.memberRemoved"))
      setRemoveTarget(null)
      fetchMembers()

    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("user.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchRemove = async () => {
    try {
      await removeWorkspaceUsers(workspaceId, Array.from(selected))
      toast.success(t("workspace.memberRemoved"))
      setBatchRemoveOpen(false)
      clearSelection()
      fetchMembers()

    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("user.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleAddSuccess = () => {
    fetchMembers()

  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("workspace.members")}</h1>
          <p className="text-muted-foreground text-sm">{t("workspace.membersManage", { count: totalCount })}</p>
        </div>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t("workspace.addMember")}
        </Button>
      </div>
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("user.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && (
          <Button variant="destructive" size="sm" onClick={() => setBatchRemoveOpen(true)}>
            <UserMinus className="mr-2 h-4 w-4" />
            {t("workspace.removeMember")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <Checkbox checked={members.length > 0 && selected.size === members.length} onCheckedChange={() => toggleAll(members.map((m) => m.metadata.id))} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("username")}>{t("user.username")}<SortIcon field="username" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("email")}>{t("user.email")}<SortIcon field="email" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>{t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("phone")}>{t("common.phone")}<SortIcon field="phone" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead>{t("user.role")}</TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("common.status")}
                      <Filter className={`h-3 w-3 ${statusFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setStatusFilter("all")}>{t("common.all")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setStatusFilter("active")}>{t("common.active")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setStatusFilter("inactive")}>{t("common.inactive")}</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>{t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>{t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} /></TableHead>
              <TableHead className="w-16">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>{Array.from({ length: 10 }).map((_, j) => (<TableCell key={j}><Skeleton className="h-4 w-20" /></TableCell>))}</TableRow>
              ))
            ) : members.length === 0 ? (
              <TableRow><TableCell colSpan={10} className="text-muted-foreground py-8 text-center">{t("workspace.noMembers")}</TableCell></TableRow>
            ) : (
              members.map((m) => (
                <TableRow key={m.metadata.id}>
                  <TableCell><Checkbox checked={selected.has(m.metadata.id)} onCheckedChange={() => toggleOne(m.metadata.id)} /></TableCell>
                  <TableCell className="font-medium">
                    <Link to={`/iam/workspaces/${workspaceId}/users/${m.metadata.id}`} className="hover:underline">
                      {m.spec.username}
                    </Link>
                  </TableCell>
                  <TableCell>{m.spec.email}</TableCell>
                  <TableCell>{m.spec.displayName || "-"}</TableCell>
                  <TableCell>{m.spec.phone || "-"}</TableCell>
                  <TableCell>{m.spec.role ? t(`role.${m.spec.role}`, { defaultValue: m.spec.role }) : "-"}</TableCell>
                  <TableCell>
                    <Badge variant={m.spec.status === "active" ? "default" : "secondary"}>
                      {m.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">{new Date(m.metadata.createdAt).toLocaleString()}</TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">{new Date(m.metadata.updatedAt).toLocaleString()}</TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setRemoveTarget(m)} title={t("workspace.removeMember")}>
                      <UserMinus className="h-3.5 w-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* pagination */}
      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* add member dialog */}
      <AddMemberDialog open={addOpen} onOpenChange={setAddOpen} workspaceId={workspaceId} existingMemberIds={members.map((m) => m.metadata.id)} onSuccess={handleAddSuccess} />

      {/* remove confirm */}
      <ConfirmDialog
        open={!!removeTarget}
        onOpenChange={(v) => { if (!v) setRemoveTarget(null) }}
        title={t("workspace.removeMember")}
        description={t("workspace.removeMemberConfirm", { name: removeTarget?.spec.username ?? "" })}
        onConfirm={handleRemove}
        confirmText={t("common.confirm")}
      />

      {/* batch remove confirm */}
      <ConfirmDialog
        open={batchRemoveOpen}
        onOpenChange={setBatchRemoveOpen}
        title={t("workspace.removeMember")}
        description={t("workspace.batchRemoveMemberConfirm", { count: selected.size })}
        onConfirm={handleBatchRemove}
        confirmText={t("common.confirm")}
      />
    </div>
  )
}

// ===== Add Member Dialog =====

function AddMemberDialog({
  open, onOpenChange, workspaceId, existingMemberIds, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspaceId: string
  existingMemberIds: string[]
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [allUsers, setAllUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = useState("")
  const [loading, setLoading] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [submitting, setSubmitting] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")

  useEffect(() => {
    if (open) {
      setSelectedIds(new Set())
      setSelectedRoleId("")
      setSearchQuery("")
      setLoading(true)
      Promise.all([
        listUsers({ pageSize: 100 }),
        listWorkspaceRoles(workspaceId, { pageSize: 100 }),
      ]).then(([userData, roleData]) => {
        setAllUsers(userData.items ?? [])
        const items = roleData.items ?? []
        setRoles(items)
        const viewer = items.find((r) => r.spec.name.includes("viewer"))
        if (viewer) setSelectedRoleId(viewer.metadata.id)
      }).finally(() => setLoading(false))
    }
  }, [open, workspaceId])

  const availableUsers = allUsers.filter((u) => !existingMemberIds.includes(u.metadata.id))

  const filteredUsers = searchQuery
    ? availableUsers.filter((u) => {
        const q = searchQuery.toLowerCase()
        return (
          u.spec.username.toLowerCase().includes(q) ||
          (u.spec.email?.toLowerCase().includes(q)) ||
          (u.spec.displayName?.toLowerCase().includes(q)) ||
          (u.spec.phone?.includes(q))
        )
      })
    : availableUsers

  const handleToggle = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const handleSubmit = async () => {
    if (selectedIds.size === 0) return
    setSubmitting(true)
    try {
      const userIds = Array.from(selectedIds)
      await addWorkspaceUsers(workspaceId, userIds)
      if (selectedRoleId) {
        await Promise.all(userIds.map((uid) =>
          createWorkspaceRoleBinding(workspaceId, { spec: { userId: uid, roleId: selectedRoleId, scope: "workspace" } })
        ))
      }
      toast.success(t("workspace.memberAdded"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("user.title") }) : err.message)
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
          <DialogTitle>{t("workspace.addMember")}</DialogTitle>
          <DialogDescription>{t("workspace.addMemberDesc")}</DialogDescription>
        </DialogHeader>
        <div>
          <p className="mb-2 text-sm font-medium">{t("rolebinding.selectRole")}</p>
          <div className="max-h-[120px] overflow-auto border rounded-md">
            {loading ? (
              <div className="space-y-2 p-4">{Array.from({ length: 2 }).map((_, i) => <Skeleton key={i} className="h-8 w-full" />)}</div>
            ) : roles.map((role) => (
              <label key={role.metadata.id} className={`flex cursor-pointer items-center gap-3 px-4 py-2 hover:bg-muted/50 ${selectedRoleId === role.metadata.id ? "bg-muted" : ""}`}>
                <Checkbox checked={selectedRoleId === role.metadata.id} onCheckedChange={() => setSelectedRoleId(role.metadata.id)} />
                <div className="flex-1">
                  <p className="text-sm font-medium">{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || role.spec.name })}</p>
                  <p className="text-muted-foreground text-xs">{t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "" })}</p>
                </div>
              </label>
            ))}
          </div>
        </div>
        <div>
          <p className="mb-2 text-sm font-medium">{t("rolebinding.selectUser")}</p>
          <div className="relative mb-2">
            <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
            <Input
              placeholder={t("user.searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
          <div className="max-h-[200px] overflow-auto border rounded-md">
          {loading ? (
            <div className="space-y-2 p-4">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-8 w-full" />)}</div>
          ) : filteredUsers.length === 0 ? (
            <p className="text-muted-foreground p-4 text-center text-sm">{searchQuery ? t("common.noSearchResults") : t("workspace.noAvailableUsers")}</p>
          ) : (
            filteredUsers.map((user) => {
              const isInactive = user.spec.status === "inactive"
              return (
                <label key={user.metadata.id} className={`flex cursor-pointer items-center gap-3 px-4 py-2 hover:bg-muted/50 ${isInactive ? "opacity-50" : ""}`}>
                  <Checkbox checked={selectedIds.has(user.metadata.id)} onCheckedChange={() => handleToggle(user.metadata.id)} />
                  <div className="flex-1">
                    <p className="text-sm font-medium">
                      {user.spec.username}
                      {isInactive && (
                        <Badge variant="secondary" className="ml-2 text-[10px]">{t("common.inactive")}</Badge>
                      )}
                    </p>
                    <p className="text-muted-foreground text-xs">{user.spec.displayName || user.spec.email}</p>
                  </div>
                </label>
              )
            })
          )}
          </div>
        </div>
        <DialogFooter className="mt-6 pt-4 border-t">
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={selectedIds.size === 0 || !selectedRoleId || submitting}>
            {submitting ? "..." : t("workspace.addMember")} {selectedIds.size > 0 && `(${selectedIds.size})`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
