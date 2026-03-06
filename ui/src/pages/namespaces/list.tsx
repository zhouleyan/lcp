import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router"
import {
  Plus, Pencil, Trash2,
  Search, Filter,
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
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  listNamespaces, createNamespace, updateNamespace, deleteNamespace, deleteNamespaces,
} from "@/api/namespaces"
import { listWorkspaces } from "@/api/workspaces"
import { ApiError, translateApiError } from "@/api/client"
import type { Namespace, Workspace, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

type SortField = "name" | "display_name" | "member_count" | "created_at" | "updated_at"

export default function NamespaceListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const [namespaces, setNamespaces] = useState<Namespace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")
  const [visibilityFilter, setVisibilityFilter] = useState("all")

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Namespace | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Namespace | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      if (visibilityFilter !== "all") params.visibility = visibilityFilter
      const data = await listNamespaces(params)
      setNamespaces(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch {
      toast.error(t("api.error.internalError"))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter, visibilityFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, visibilityFilter, pageSize])
  useEffect(() => { clearSelection() }, [namespaces])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteNamespace(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("namespace.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await deleteNamespaces(Array.from(selected))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("namespace.title") }) : err.message)
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
          <h1 className="text-2xl font-bold">{t("namespace.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("namespace.manage", { count: totalCount })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {selected.size > 0 && (
            <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("namespace.batchDelete")} ({selected.size})
            </Button>
          )}
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("namespace.create")}
          </Button>
        </div>
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("namespace.searchPlaceholder")}
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
                  checked={namespaces.length > 0 && selected.size === namespaces.length}
                  onCheckedChange={() => toggleAll(namespaces.map((ns) => ns.metadata.id))}
                />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("namespace.workspaceName")}</TableHead>
              <TableHead>{t("namespace.owner")}</TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("namespace.visibility")}
                      <Filter className={`h-3 w-3 ${visibilityFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setVisibilityFilter("all")}>{t("common.all")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setVisibilityFilter("public")}>{t("namespace.visibility.public")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setVisibilityFilter("private")}>{t("namespace.visibility.private")}</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
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
              <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("member_count")}>
                {t("namespace.memberCount")}<SortIcon field="member_count" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 10 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : namespaces.length === 0 ? (
              <TableRow>
                <TableCell colSpan={10} className="text-muted-foreground py-8 text-center">
                  {t("namespace.noData")}
                </TableCell>
              </TableRow>
            ) : (
              namespaces.map((ns) => (
                <TableRow key={ns.metadata.id}>
                  <TableCell>
                    <Checkbox
                      checked={selected.has(ns.metadata.id)}
                      onCheckedChange={() => toggleOne(ns.metadata.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <Link to={`/namespaces/${ns.metadata.id}`} className="font-medium hover:underline">
                      {ns.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{ns.spec.displayName || "-"}</TableCell>
                  <TableCell className="text-sm">
                    <Link to={`/workspaces/${ns.spec.workspaceId}`} className="hover:underline">
                      {ns.spec.workspaceName || ns.spec.workspaceId}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm">{ns.spec.ownerName || ns.spec.ownerId}</TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {ns.spec.visibility === "private" ? t("namespace.visibility.private") : t("namespace.visibility.public")}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={ns.spec.status === "active" ? "default" : "secondary"}>
                      {ns.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-center">{ns.spec.memberCount ?? 0}/{ns.spec.maxMembers || "∞"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ns.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(ns)} title={t("common.edit")}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(ns)} title={t("common.delete")}>
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
      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* create dialog */}
      <NamespaceFormDialog open={createOpen} onOpenChange={setCreateOpen} onSuccess={fetchData} />

      {/* edit dialog */}
      <NamespaceFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        namespace={editTarget ?? undefined}
        onSuccess={fetchData}
      />

      {/* delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("namespace.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      {/* batch delete confirm */}
      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("namespace.batchDelete")}
        description={t("namespace.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Namespace Form Dialog =====

interface NamespaceFormValues {
  name: string
  workspaceId: string
  displayName: string
  description: string
  visibility: "public" | "private"
  status: "active" | "inactive"
  maxMembers: number
}

function NamespaceFormDialog({
  open, onOpenChange, namespace, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  namespace?: Namespace
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!namespace
  const [loading, setLoading] = useState(false)
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [workspacesLoading, setWorkspacesLoading] = useState(false)

  useEffect(() => {
    if (open && !isEdit) {
      setWorkspacesLoading(true)
      listWorkspaces({ page: 1, pageSize: 100, status: "active" })
        .then((data) => setWorkspaces(data.items ?? []))
        .catch(() => {})
        .finally(() => setWorkspacesLoading(false))
    }
  }, [open, isEdit])

  const schema = z.object({
    name: z.string()
      .min(3, t("namespace.validation.name.format"))
      .max(50, t("namespace.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("namespace.validation.name.format")),
    workspaceId: z.string()
      .min(1, t("api.validation.required", { field: t("namespace.workspaceName") })),
    displayName: z.string().optional(),
    description: z.string().optional(),
    visibility: z.enum(["public", "private"]),
    status: z.enum(["active", "inactive"]),
    maxMembers: z.number().int().min(0, t("namespace.validation.maxMembers")),
  })

  const form = useForm<NamespaceFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", workspaceId: "", displayName: "", description: "", visibility: "public", status: "active", maxMembers: 0 },
  })

  useEffect(() => {
    if (open) {
      if (namespace) {
        form.reset({
          name: namespace.metadata.name,
          workspaceId: namespace.spec.workspaceId ?? "",
          displayName: namespace.spec.displayName ?? "",
          description: namespace.spec.description ?? "",
          visibility: namespace.spec.visibility ?? "public",
          status: namespace.spec.status ?? "active",
          maxMembers: namespace.spec.maxMembers ?? 0,
        })
      } else {
        form.reset({ name: "", workspaceId: "", displayName: "", description: "", visibility: "public", status: "active", maxMembers: 0 })
      }
    }
  }, [open, namespace, form])

  const checkUniqueness = async (value: string) => {
    if (!value || isEdit) return
    try {
      const data = await listNamespaces({ page: 1, pageSize: 1, search: value })
      const exists = data.items?.some((ns) => ns.metadata.name === value)
      if (exists) form.setError("name", { message: t("namespace.validation.name.taken") })
    } catch { /* backend will enforce */ }
  }

  const onSubmit = async (values: NamespaceFormValues) => {
    setLoading(true)
    try {
      if (isEdit) {
        await updateNamespace(namespace.metadata.id, {
          metadata: namespace.metadata,
          spec: {
            ...namespace.spec,
            displayName: values.displayName,
            description: values.description,
            visibility: values.visibility,
            status: values.status,
            maxMembers: values.maxMembers,
          },
        })
        toast.success(t("action.updateSuccess"))
      } else {
        await createNamespace({
          metadata: { name: values.name } as Namespace["metadata"],
          spec: {
            workspaceId: values.workspaceId,
            displayName: values.displayName,
            description: values.description,
            visibility: values.visibility,
            status: values.status,
            maxMembers: values.maxMembers,
          } as Namespace["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        form.setError("root", { message: translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("namespace.title") }) : err.message })
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
          <DialogTitle>{isEdit ? t("namespace.edit") : t("namespace.create")}</DialogTitle>
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
                      placeholder="my-namespace"
                      onBlur={async (e) => {
                        field.onBlur()
                        if (!e.target.value) return
                        const valid = await form.trigger("name")
                        if (valid) checkUniqueness(e.target.value)
                      }}
                    />
                  </FormControl>
                  {!isEdit && <p className="text-muted-foreground text-xs">{t("namespace.validation.name.hint")}</p>}
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="workspaceId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("namespace.workspaceName")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange} disabled={isEdit || workspacesLoading}>
                    <FormControl>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder={workspacesLoading ? "..." : t("namespace.selectWorkspace")} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {workspaces.map((ws) => (
                        <SelectItem key={ws.metadata.id} value={ws.metadata.id}>
                          {ws.spec.displayName || ws.metadata.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
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
              name="visibility"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("namespace.visibility")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="public">{t("namespace.visibility.public")}</SelectItem>
                      <SelectItem value="private">{t("namespace.visibility.private")}</SelectItem>
                    </SelectContent>
                  </Select>
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
            <FormField
              control={form.control}
              name="maxMembers"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("namespace.maxMembers")}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={0}
                      {...field}
                      onChange={(e) => field.onChange(e.target.value === "" ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <p className="text-muted-foreground text-xs">{t("namespace.maxMembersHint")}</p>
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
