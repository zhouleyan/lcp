import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router"
import { Plus, Pencil, Trash2, Search } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  listNetworks, createNetwork, updateNetwork, deleteNetwork, deleteNetworks,
} from "@/api/network/networks"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Network, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { cidrUsableRange } from "./utils"

export default function NetworkListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const [networks, setNetworks] = useState<Network[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Network | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Network | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search

      const data = await listNetworks(params)
      setNetworks(data.items ?? [])
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
  useEffect(() => { clearSelection() }, [networks])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteNetwork(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "network.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteNetworks(ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "network.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("network.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("network.manage", { count: totalCount })}
          </p>
        </div>
        {hasPermission("network:networks:create") && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("network.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("network.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission("network:networks:deleteCollection") && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("network.deleteSelected")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {hasPermission("network:networks:deleteCollection") && (
                  <Checkbox
                    checked={networks.length > 0 && selected.size === networks.length}
                    onCheckedChange={() => toggleAll(networks.map((n) => n.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("cidr")}>
                {t("network.cidr")}<SortIcon field="cidr" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("network.isPublic")}</TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("subnet_count")}>
                {t("network.subnetCount")}<SortIcon field="subnet_count" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                {t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-28">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 9 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : networks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-muted-foreground py-8 text-center">
                  {t("network.noData")}
                </TableCell>
              </TableRow>
            ) : (
              networks.map((net) => (
                <TableRow key={net.metadata.id}>
                  <TableCell>
                    {hasPermission("network:networks:deleteCollection") && (
                      <Checkbox
                        checked={selected.has(net.metadata.id)}
                        onCheckedChange={() => toggleOne(net.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Link to={`${net.metadata.id}`} className="font-medium hover:underline">
                      {net.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm">{net.spec.displayName || "-"}</TableCell>
                  <TableCell className="text-sm font-mono">
                    <div>{net.spec.cidr || "-"}</div>
                    {net.spec.cidr && cidrUsableRange(net.spec.cidr) && (
                      <div className="text-xs text-muted-foreground">{cidrUsableRange(net.spec.cidr)}</div>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={net.spec.isPublic !== false ? "default" : "secondary"}>
                      {net.spec.isPublic !== false ? t("network.public") : t("network.private")}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {(() => {
                      const used = net.spec.subnetCount ?? 0
                      const total = net.spec.maxSubnets ?? 10
                      const pct = total > 0 ? Math.round((used / total) * 100) : 0
                      const barColor = pct > 90 ? "bg-primary" : pct > 60 ? "bg-primary/50" : "bg-primary/20"
                      return (
                        <div className="pr-8 space-y-1">
                          <div className="h-2 rounded-full bg-muted">
                            <div className={`h-2 rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
                          </div>
                          <div className="flex justify-between text-xs text-muted-foreground">
                            <span>{used} / {total}</span>
                            <span>{pct}%</span>
                          </div>
                        </div>
                      )
                    })()}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(net.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {net.metadata.updatedAt ? new Date(net.metadata.updatedAt).toLocaleString() : "-"}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="h-8 px-2">
                          {t("common.actions")}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {hasPermission("network:networks:update") && (
                          <DropdownMenuItem onClick={() => setEditTarget(net)}>
                            <Pencil className="mr-2 h-3.5 w-3.5" />
                            {t("common.edit")}
                          </DropdownMenuItem>
                        )}
                        {hasPermission("network:networks:delete") && (
                          <>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(net)}>
                              <Trash2 className="mr-2 h-3.5 w-3.5" />
                              {t("common.delete")}
                            </DropdownMenuItem>
                          </>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* Create dialog */}
      <NetworkFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
      />

      {/* Edit dialog */}
      <NetworkFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        network={editTarget ?? undefined}
        onSuccess={fetchData}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("network.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("network.deleteSelected")}
        description={t("network.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Network Form Dialog =====

interface NetworkFormValues {
  name: string
  displayName: string
  description: string
  cidr: string
  maxSubnets: number
  isPublic: boolean
}

function NetworkFormDialog({
  open, onOpenChange, network, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  network?: Network
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!network
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.networkFormat"))
      .max(50, t("api.validation.name.networkFormat"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.networkFormat")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    cidr: z.string().optional().refine(
      (v) => !v || /^\d{1,3}(\.\d{1,3}){3}\/\d{1,2}$/.test(v),
      t("api.validation.cidr.format"),
    ),
    maxSubnets: z.coerce.number().int().min(1, t("network.maxSubnetsRange")).max(50, t("network.maxSubnetsRange")),
    isPublic: z.boolean(),
  })

  const form = useForm<NetworkFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", displayName: "", description: "", cidr: "", maxSubnets: 10, isPublic: true },
  })

  useEffect(() => {
    if (open) {
      if (network) {
        form.reset({
          name: network.metadata.name,
          displayName: network.spec.displayName ?? "",
          description: network.spec.description ?? "",
          cidr: network.spec.cidr ?? "",
          maxSubnets: network.spec.maxSubnets ?? 10,
          isPublic: network.spec.isPublic !== false,
        })
      } else {
        form.reset({ name: "", displayName: "", description: "", cidr: "", maxSubnets: 10, isPublic: true })
      }
    }
  }, [open, network, form])

  const onSubmit = async (values: NetworkFormValues) => {
    setLoading(true)
    try {
      const spec: Network["spec"] = {
        displayName: values.displayName,
        description: values.description,
        cidr: values.cidr || undefined,
        maxSubnets: values.maxSubnets,
        isPublic: values.isPublic,
      }

      const payload = {
        metadata: isEdit ? network.metadata : { name: values.name } as Network["metadata"],
        spec,
      }

      if (isEdit) {
        await updateNetwork(network.metadata.id, payload)
        toast.success(t("action.updateSuccess"))
      } else {
        await createNetwork(payload)
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const fieldName = d.field.replace(/^(spec|metadata)\./, "") as keyof NetworkFormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(fieldName, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`network.${fieldName}`) || fieldName }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("network.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} onCloseAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("network.edit") : t("network.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col flex-1 overflow-hidden">
            {form.formState.errors.root && (
              <div className="shrink-0 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
            <FormField control={form.control} name="name" render={({ field }) => (
              <FormItem>
                <FormLabel required>{t("network.name")}</FormLabel>
                <FormControl><Input {...field} disabled={isEdit} placeholder="my-network" /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("network.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("network.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="cidr" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("network.cidr")}</FormLabel>
                <FormControl><Input {...field} disabled={isEdit} placeholder={t("network.cidrPlaceholder")} className="font-mono" /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="maxSubnets" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("network.maxSubnets")}</FormLabel>
                <FormControl><Input type="number" min={1} max={50} placeholder={t("network.maxSubnetsPlaceholder")} {...field} disabled={isEdit} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="isPublic" render={({ field }) => (
              <FormItem className="flex items-center justify-between rounded-lg border p-3">
                <FormLabel className="cursor-pointer">{t("network.isPublic")}</FormLabel>
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                </FormControl>
              </FormItem>
            )} />
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
