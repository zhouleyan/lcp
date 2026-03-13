import { useCallback, useEffect, useState } from "react"
import { Plus, Pencil, Trash2, Search, Filter } from "lucide-react"
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
  listEndpoints, createEndpoint, updateEndpoint,
  deleteEndpoint, deleteEndpoints,
} from "@/api/o11y/endpoints"
import { showApiError } from "@/api/client"
import type { Endpoint, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

export default function EndpointListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const [endpoints, setEndpoints] = useState<Endpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Endpoint | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Endpoint | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const permPrefix = "o11y:endpoints"

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter

      const data = await listEndpoints(params)
      setEndpoints(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])
  useEffect(() => { clearSelection() }, [endpoints])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteEndpoint(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "endpoint.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteEndpoints(ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "endpoint.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("endpoint.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("common.total", { count: totalCount })}
          </p>
        </div>
        {hasPermission(`${permPrefix}:create`) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("endpoint.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("endpoint.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("endpoint.deleteSelected")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {hasPermission(`${permPrefix}:deleteCollection`) && (
                  <Checkbox
                    checked={endpoints.length > 0 && selected.size === endpoints.length}
                    onCheckedChange={() => toggleAll(endpoints.map((e) => e.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("common.description")}</TableHead>
              <TableHead>{t("endpoint.metricsUrl")}</TableHead>
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
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : endpoints.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-muted-foreground py-8 text-center">
                  {t("endpoint.noData")}
                </TableCell>
              </TableRow>
            ) : (
              endpoints.map((ep) => (
                <TableRow key={ep.metadata.id}>
                  <TableCell>
                    {hasPermission(`${permPrefix}:deleteCollection`) && (
                      <Checkbox
                        checked={selected.has(ep.metadata.id)}
                        onCheckedChange={() => toggleOne(ep.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell className="font-medium">{ep.metadata.name}</TableCell>
                  <TableCell className="max-w-[200px] truncate text-muted-foreground text-sm" title={ep.spec.description}>
                    {ep.spec.description || "-"}
                  </TableCell>
                  <TableCell className="max-w-[250px] truncate text-muted-foreground text-sm" title={ep.spec.metricsUrl}>
                    {ep.spec.metricsUrl || "-"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={ep.spec.status === "active" ? "default" : "secondary"}>
                      {ep.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ep.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {hasPermission(`${permPrefix}:update`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(ep)} title={t("common.edit")}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {hasPermission(`${permPrefix}:delete`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(ep)} title={t("common.delete")}>
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      <EndpointFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
      />

      <EndpointFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        endpoint={editTarget ?? undefined}
        onSuccess={fetchData}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("endpoint.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("endpoint.deleteSelected")}
        description={t("endpoint.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Endpoint Form Dialog =====

interface EndpointFormValues {
  name: string
  description: string
  metricsUrl: string
  status: "active" | "inactive"
}

function EndpointFormDialog({
  open, onOpenChange, endpoint, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  endpoint?: Endpoint
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!endpoint
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
    description: z.string().optional(),
    metricsUrl: z.string().min(1, t("api.validation.required", { field: t("endpoint.metricsUrl") })),
    status: z.enum(["active", "inactive"]),
  })

  const form = useForm<EndpointFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", description: "", metricsUrl: "", status: "active" },
  })

  useEffect(() => {
    if (open) {
      if (endpoint) {
        form.reset({
          name: endpoint.metadata.name,
          description: endpoint.spec.description ?? "",
          metricsUrl: endpoint.spec.metricsUrl ?? "",
          status: (endpoint.spec.status as "active" | "inactive") ?? "active",
        })
      } else {
        form.reset({ name: "", description: "", metricsUrl: "", status: "active" })
      }
    }
  }, [open, endpoint, form])

  const onSubmit = async (values: EndpointFormValues) => {
    setLoading(true)
    try {
      const payload = {
        metadata: { name: values.name } as Endpoint["metadata"],
        spec: {
          description: values.description,
          metricsUrl: values.metricsUrl,
          status: values.status,
        } as Endpoint["spec"],
      }

      if (isEdit) {
        payload.metadata = endpoint.metadata
        await updateEndpoint(endpoint.metadata.id, payload)
        toast.success(t("action.updateSuccess"))
      } else {
        await createEndpoint(payload)
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      showApiError(err, t, "endpoint.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("endpoint.edit") : t("endpoint.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col flex-1 overflow-hidden">
            {form.formState.errors.root && (
              <div className="shrink-0 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t("common.name")}</FormLabel>
                  <FormControl>
                    <Input {...field} disabled={isEdit} placeholder="my-endpoint" />
                  </FormControl>
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
              name="metricsUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t("endpoint.metricsUrl")}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t("endpoint.metricsUrlPlaceholder")} />
                  </FormControl>
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
            </div>
            <DialogFooter className="mt-6 pt-4 border-t shrink-0">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
