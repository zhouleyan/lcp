import { useCallback, useEffect, useState } from "react"
import { Plus, Pencil, Trash2, Search, Activity, CheckCircle2, XCircle, Loader2 } from "lucide-react"
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
import { Switch } from "@/components/ui/switch"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover"
import {
  listEndpoints, createEndpoint, updateEndpoint,
  deleteEndpoint, deleteEndpoints, probeEndpoint,
} from "@/api/o11y/endpoints"
import { showApiError } from "@/api/client"
import type { Endpoint, ListParams, ProbeResultItem } from "@/api/types"
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

      const data = await listEndpoints(params)
      setEndpoints(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, pageSize])
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
              <TableHead>{t("endpoint.endpoints")}</TableHead>
              <TableHead>{t("endpoint.visibility")}</TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                {t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 8 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : endpoints.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="text-muted-foreground py-8 text-center">
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
                  <TableCell className="text-muted-foreground text-sm">
                    <div className="space-y-0.5">
                      {[
                        { label: t("endpoint.metricsLabel"), url: ep.spec.metricsUrl },
                        { label: t("endpoint.logsLabel"), url: ep.spec.logsUrl },
                        { label: t("endpoint.tracesLabel"), url: ep.spec.tracesUrl },
                        { label: t("endpoint.apmLabel"), url: ep.spec.apmUrl },
                      ].filter((e) => e.url).map((e) => (
                        <div key={e.label} className="flex items-center gap-1.5 max-w-[350px]">
                          <span className="shrink-0 text-xs font-medium text-foreground w-12">{e.label}</span>
                          <span className="truncate" title={e.url}>{e.url}</span>
                        </div>
                      ))}
                      {!ep.spec.metricsUrl && !ep.spec.logsUrl && !ep.spec.tracesUrl && !ep.spec.apmUrl && "-"}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={ep.spec.isPublic !== false ? "default" : "secondary"}>
                      {ep.spec.isPublic !== false ? t("endpoint.public") : t("endpoint.private")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ep.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(ep.metadata.updatedAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {hasPermission(`${permPrefix}:probe`) && (
                        <ProbeButton endpointId={ep.metadata.id} />
                      )}
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
  isPublic: boolean
  metricsUrl: string
  logsUrl: string
  tracesUrl: string
  apmUrl: string
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
      .regex(/^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$/, t("api.validation.name.format")),
    description: z.string().optional(),
    isPublic: z.boolean(),
    metricsUrl: z.string()
      .min(1, t("api.validation.required", { field: t("endpoint.metricsUrl") }))
      .url(t("endpoint.urlInvalid")),
    logsUrl: z.string().refine((v) => !v || z.string().url().safeParse(v).success, t("endpoint.urlInvalid")).optional(),
    tracesUrl: z.string().refine((v) => !v || z.string().url().safeParse(v).success, t("endpoint.urlInvalid")).optional(),
    apmUrl: z.string().refine((v) => !v || z.string().url().safeParse(v).success, t("endpoint.urlInvalid")).optional(),
  })

  const form = useForm<EndpointFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", description: "", isPublic: true, metricsUrl: "", logsUrl: "", tracesUrl: "", apmUrl: "" },
  })

  useEffect(() => {
    if (open) {
      if (endpoint) {
        form.reset({
          name: endpoint.metadata.name,
          description: endpoint.spec.description ?? "",
          isPublic: endpoint.spec.isPublic !== false,
          metricsUrl: endpoint.spec.metricsUrl ?? "",
          logsUrl: endpoint.spec.logsUrl ?? "",
          tracesUrl: endpoint.spec.tracesUrl ?? "",
          apmUrl: endpoint.spec.apmUrl ?? "",
        })
      } else {
        form.reset({ name: "", description: "", isPublic: true, metricsUrl: "", logsUrl: "", tracesUrl: "", apmUrl: "" })
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
          isPublic: values.isPublic,
          metricsUrl: values.metricsUrl,
          logsUrl: values.logsUrl,
          tracesUrl: values.tracesUrl,
          apmUrl: values.apmUrl,
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
              name="logsUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("endpoint.logsUrl")}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t("endpoint.urlPlaceholder")} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="tracesUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("endpoint.tracesUrl")}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t("endpoint.urlPlaceholder")} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apmUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("endpoint.apmUrl")}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t("endpoint.urlPlaceholder")} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="isPublic"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between">
                  <FormLabel className="cursor-pointer">{t("endpoint.visibility")}</FormLabel>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">
                      {field.value ? t("endpoint.public") : t("endpoint.private")}
                    </span>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </div>
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

// ===== Probe Button =====

const fieldLabelKeys: Record<string, string> = {
  metricsUrl: "endpoint.metricsLabel",
  logsUrl: "endpoint.logsLabel",
  tracesUrl: "endpoint.tracesLabel",
  apmUrl: "endpoint.apmLabel",
}

function ProbeButton({ endpointId }: { endpointId: string }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState<ProbeResultItem[] | null>(null)
  const [open, setOpen] = useState(false)

  const handleProbe = async () => {
    setLoading(true)
    setResults(null)
    setOpen(true)
    try {
      const data = await probeEndpoint(endpointId)
      setResults(data.results)
    } catch (err) {
      showApiError(err, t, "endpoint.title")
      setOpen(false)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleProbe} title={t("endpoint.probe")}>
          {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Activity className="h-3.5 w-3.5" />}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="end">
        <div className="space-y-2">
          <p className="text-sm font-medium">{t("endpoint.probeResult")}</p>
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("endpoint.probing")}
            </div>
          )}
          {results?.map((item) => (
            <div key={item.field} className="flex items-start gap-2 text-sm">
              {item.success
                ? <CheckCircle2 className="h-4 w-4 shrink-0 text-green-500 mt-0.5" />
                : <XCircle className="h-4 w-4 shrink-0 text-destructive mt-0.5" />
              }
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className="font-medium">{t(fieldLabelKeys[item.field] ?? item.field)}</span>
                  {item.success
                    ? <span className="text-green-600">{item.statusCode} · {item.duration}</span>
                    : <span className="text-destructive">{item.phase} · {item.duration}</span>
                  }
                </div>
                <div className="truncate text-muted-foreground" title={item.url}>{item.url}</div>
                {item.message && <div className="text-destructive text-xs">{item.message}</div>}
              </div>
            </div>
          ))}
          {results?.length === 0 && (
            <p className="text-sm text-muted-foreground">{t("endpoint.probeNoUrls")}</p>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
