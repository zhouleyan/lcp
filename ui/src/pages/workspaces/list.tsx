import { useCallback, useEffect, useRef, useState } from "react"
import { Link } from "react-router"
import {
  Plus, Pencil, Trash2, ArrowUpDown, ArrowUp, ArrowDown,
  Search, Filter, ChevronLeft, ChevronRight,
} from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  listWorkspaces, createWorkspace, updateWorkspace, deleteWorkspace, deleteWorkspaces,
} from "@/api/workspaces"
import { ApiError, translateApiError } from "@/api/client"
import type { Workspace, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
type SortField = "name" | "display_name" | "namespace_count" | "member_count" | "created_at" | "updated_at"

export default function WorkspaceListPage() {
  const { t } = useTranslation()
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [sortBy, setSortBy] = useState<SortField>("created_at")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc")
  const [searchInput, setSearchInput] = useState("")
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Workspace | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Workspace | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))

  // Debounce search
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(null)
  useEffect(() => {
    searchTimer.current = setTimeout(() => setSearch(searchInput), 300)
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current) }
  }, [searchInput])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      const data = await listWorkspaces(params)
      setWorkspaces(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch {
      toast.error(t("api.error.internalError"))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])
  useEffect(() => { setSelected(new Set()) }, [workspaces])

  const handleSort = (field: SortField) => {
    if (sortBy === field) {
      setSortOrder((o) => (o === "asc" ? "desc" : "asc"))
    } else {
      setSortBy(field)
      setSortOrder("asc")
    }
  }

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortBy !== field) return <ArrowUpDown className="ml-1 inline h-3 w-3 opacity-40" />
    return sortOrder === "asc"
      ? <ArrowUp className="ml-1 inline h-3 w-3" />
      : <ArrowDown className="ml-1 inline h-3 w-3" />
  }

  const toggleAll = () => {
    setSelected(selected.size === workspaces.length ? new Set() : new Set(workspaces.map((w) => w.metadata.id)))
  }
  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteWorkspace(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("workspace.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await deleteWorkspaces(Array.from(selected))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      setSelected(new Set())
      fetchData()
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("workspace.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("workspace.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("workspace.manage", { count: totalCount })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {selected.size > 0 && (
            <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("workspace.batchDelete")} ({selected.size})
            </Button>
          )}
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("workspace.create")}
          </Button>
        </div>
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("workspace.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <Checkbox
                  checked={workspaces.length > 0 && selected.size === workspaces.length}
                  onCheckedChange={toggleAll}
                />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                {t("common.displayName")}<SortIcon field="display_name" />
              </TableHead>
              <TableHead>{t("common.description")}</TableHead>
              <TableHead>{t("workspace.owner")}</TableHead>
              <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("namespace_count")}>
                {t("workspace.namespaceCount")}<SortIcon field="namespace_count" />
              </TableHead>
              <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("member_count")}>
                {t("workspace.memberCount")}<SortIcon field="member_count" />
              </TableHead>
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
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                {t("common.updated")}<SortIcon field="updated_at" />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 11 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : workspaces.length === 0 ? (
              <TableRow>
                <TableCell colSpan={11} className="text-muted-foreground py-8 text-center">
                  {t("workspace.noData")}
                </TableCell>
              </TableRow>
            ) : (
              workspaces.map((ws) => (
                <TableRow key={ws.metadata.id}>
                  <TableCell>
                    <Checkbox
                      checked={selected.has(ws.metadata.id)}
                      onCheckedChange={() => toggleOne(ws.metadata.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <Link to={`/workspaces/${ws.metadata.id}`} className="font-medium hover:underline">
                      {ws.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{ws.spec.displayName || "-"}</TableCell>
                  <TableCell className="max-w-[200px] truncate text-muted-foreground text-sm" title={ws.spec.description}>
                    {ws.spec.description || "-"}
                  </TableCell>
                  <TableCell className="text-sm">{ws.spec.ownerName || ws.spec.ownerId}</TableCell>
                  <TableCell className="text-center">{ws.spec.namespaceCount ?? 0}</TableCell>
                  <TableCell className="text-center">{ws.spec.memberCount ?? 0}</TableCell>
                  <TableCell>
                    <Badge variant={ws.spec.status === "active" ? "default" : "secondary"}>
                      {ws.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ws.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ws.metadata.updatedAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(ws)} title={t("common.edit")}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(ws)} title={t("common.delete")}>
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* pagination */}
      {totalCount > 0 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <p className="text-muted-foreground text-sm">{t("common.total", { count: totalCount })}</p>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground text-sm">{t("common.pageSize")}</span>
              <Select value={String(pageSize)} onValueChange={(v) => setPageSize(Number(v))}>
                <SelectTrigger className="h-8 w-[70px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PAGE_SIZE_OPTIONS.map((s) => <SelectItem key={s} value={String(s)}>{s}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <Button variant="outline" size="icon" className="h-8 w-8" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm px-2">{t("common.page", { page, total: totalPages })}</span>
            <Button variant="outline" size="icon" className="h-8 w-8" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* create dialog */}
      <WorkspaceFormDialog open={createOpen} onOpenChange={setCreateOpen} onSuccess={fetchData} />

      {/* edit dialog */}
      <WorkspaceFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        workspace={editTarget ?? undefined}
        onSuccess={fetchData}
      />

      {/* delete confirm */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.delete")}</DialogTitle>
            <DialogDescription>
              {t("workspace.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleDelete}>{t("common.delete")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* batch delete confirm */}
      <Dialog open={batchDeleteOpen} onOpenChange={setBatchDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("workspace.batchDelete")}</DialogTitle>
            <DialogDescription>
              {t("workspace.batchDeleteConfirm", { count: selected.size })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBatchDeleteOpen(false)}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleBatchDelete}>{t("common.delete")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ===== Workspace Form Dialog =====

interface WorkspaceFormValues {
  name: string
  displayName: string
  description: string
  status: "active" | "inactive"
}

function WorkspaceFormDialog({
  open, onOpenChange, workspace, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspace?: Workspace
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!workspace
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("workspace.validation.name.format"))
      .max(50, t("workspace.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("workspace.validation.name.format")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    status: z.enum(["active", "inactive"]),
  })

  const form = useForm<WorkspaceFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", displayName: "", description: "", status: "active" },
  })

  useEffect(() => {
    if (open) {
      if (workspace) {
        form.reset({
          name: workspace.metadata.name,
          displayName: workspace.spec.displayName ?? "",
          description: workspace.spec.description ?? "",
          status: workspace.spec.status ?? "active",
        })
      } else {
        form.reset({ name: "", displayName: "", description: "", status: "active" })
      }
    }
  }, [open, workspace, form])

  const checkUniqueness = async (value: string) => {
    if (!value || isEdit) return
    try {
      const data = await listWorkspaces({ page: 1, pageSize: 1, search: value })
      const exists = data.items?.some((w) => w.metadata.name === value)
      if (exists) form.setError("name", { message: t("workspace.validation.name.taken") })
    } catch { /* backend will enforce */ }
  }

  const onSubmit = async (values: WorkspaceFormValues) => {
    setLoading(true)
    try {
      if (isEdit) {
        await updateWorkspace(workspace.metadata.id, {
          metadata: workspace.metadata,
          spec: { ...workspace.spec, displayName: values.displayName, description: values.description, status: values.status },
        })
        toast.success(t("action.updateSuccess"))
      } else {
        await createWorkspace({
          metadata: { name: values.name } as Workspace["metadata"],
          spec: { displayName: values.displayName, description: values.description, status: values.status } as Workspace["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        form.setError("root", { message: translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("workspace.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("workspace.edit") : t("workspace.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.name")}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      disabled={isEdit}
                      placeholder="my-workspace"
                      onBlur={async (e) => {
                        field.onBlur()
                        if (!e.target.value) return
                        const valid = await form.trigger("name")
                        if (valid) checkUniqueness(e.target.value)
                      }}
                    />
                  </FormControl>
                  {!isEdit && <p className="text-muted-foreground text-xs">{t("workspace.validation.name.hint")}</p>}
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="displayName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.displayName")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.description")}</FormLabel>
                  <FormControl><Textarea rows={3} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="status"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.status")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="active">{t("common.active")}</SelectItem>
                      <SelectItem value="inactive">{t("common.inactive")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
