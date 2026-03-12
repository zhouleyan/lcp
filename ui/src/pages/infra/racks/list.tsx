import { useCallback, useEffect, useState } from "react"
import { Link, useSearchParams } from "react-router"
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
  listRacks, createRack, updateRack, deleteRack, deleteRacks,
} from "@/api/infra/racks"
import { listLocations } from "@/api/infra/locations"
import { listSites } from "@/api/infra/sites"
import { listRegions } from "@/api/infra/regions"
import { showApiError, handleFormApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Rack, Location, Site, Region, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

export default function RackListPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const [racks, setRacks] = useState<Rack[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")
  const [regionFilter, setRegionFilter] = useState("all")
  const [siteFilter, setSiteFilter] = useState("all")
  const [locationFilter, setLocationFilter] = useState(() => searchParams.get("locationId") ?? "all")
  const [allRegions, setAllRegions] = useState<Region[]>([])
  const [allSites, setAllSites] = useState<Site[]>([])
  const [allLocations, setAllLocations] = useState<Location[]>([])

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Rack | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Rack | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const permPrefix = "infra:racks"

  // Fetch regions on mount
  useEffect(() => {
    listRegions({ page: 1, pageSize: SELECT_PAGE_SIZE }).then(data => setAllRegions(data.items ?? [])).catch(() => {})
  }, [])

  // Fetch sites when region filter changes (cascade)
  useEffect(() => {
    const params: ListParams = { page: 1, pageSize: SELECT_PAGE_SIZE }
    if (regionFilter !== "all") params.regionId = regionFilter
    listSites(params).then(data => setAllSites(data.items ?? [])).catch(() => {})
    setSiteFilter("all") // reset site filter when region changes
  }, [regionFilter])

  // Fetch locations when site filter changes (cascade)
  useEffect(() => {
    const params: ListParams = { page: 1, pageSize: SELECT_PAGE_SIZE }
    if (siteFilter !== "all") params.siteId = siteFilter
    if (regionFilter !== "all") params.regionId = regionFilter
    listLocations(params).then(data => setAllLocations(data.items ?? [])).catch(() => {})
    setLocationFilter("all") // reset location filter when site changes
  }, [siteFilter, regionFilter])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      if (locationFilter !== "all") params.locationId = locationFilter
      if (siteFilter !== "all") params.siteId = siteFilter
      if (regionFilter !== "all") params.regionId = regionFilter
      const data = await listRacks(params)
      setRacks(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter, locationFilter, siteFilter, regionFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, locationFilter, siteFilter, regionFilter, pageSize])
  useEffect(() => { clearSelection() }, [racks])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteRack(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "rack.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteRacks(ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "rack.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("rack.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("rack.manage", { count: totalCount })}
          </p>
        </div>
        {hasPermission(`${permPrefix}:create`) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("rack.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("rack.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("rack.deleteSelected")} ({selected.size})
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
                    checked={racks.length > 0 && selected.size === racks.length}
                    onCheckedChange={() => toggleAll(racks.map((r) => r.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("common.displayName")}</TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("rack.locationName")}
                      <Filter className={`h-3 w-3 ${locationFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setLocationFilter("all")}>{t("rack.filter.locationAll")}</DropdownMenuItem>
                    {allLocations.map((l) => (
                      <DropdownMenuItem key={l.metadata.id} onClick={() => setLocationFilter(l.metadata.id)}>
                        {l.spec.displayName || l.metadata.name}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("rack.siteName")}
                      <Filter className={`h-3 w-3 ${siteFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setSiteFilter("all")}>{t("rack.filter.siteAll")}</DropdownMenuItem>
                    {allSites.map((s) => (
                      <DropdownMenuItem key={s.metadata.id} onClick={() => setSiteFilter(s.metadata.id)}>
                        {s.spec.displayName || s.metadata.name}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("rack.regionName")}
                      <Filter className={`h-3 w-3 ${regionFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setRegionFilter("all")}>{t("rack.filter.regionAll")}</DropdownMenuItem>
                    {allRegions.map((r) => (
                      <DropdownMenuItem key={r.metadata.id} onClick={() => setRegionFilter(r.metadata.id)}>
                        {r.spec.displayName || r.metadata.name}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead className="text-center">{t("rack.uHeight")}</TableHead>
              <TableHead>{t("rack.position")}</TableHead>
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
                  {Array.from({ length: 11 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : racks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={11} className="text-muted-foreground py-8 text-center">
                  {t("rack.noData")}
                </TableCell>
              </TableRow>
            ) : (
              racks.map((rack) => (
                <TableRow key={rack.metadata.id}>
                  <TableCell>
                    {hasPermission(`${permPrefix}:deleteCollection`) && (
                      <Checkbox
                        checked={selected.has(rack.metadata.id)}
                        onCheckedChange={() => toggleOne(rack.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Link to={`${rack.metadata.id}`} className="font-medium hover:underline">
                      {rack.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{rack.spec.displayName || "-"}</TableCell>
                  <TableCell>
                    {rack.spec.locationId ? (
                      <Link to={`/infra/locations/${rack.spec.locationId}`} className="hover:underline">
                        {rack.spec.locationName || rack.spec.locationId}
                      </Link>
                    ) : "-"}
                  </TableCell>
                  <TableCell>
                    {rack.spec.siteId ? (
                      <Link to={`/infra/sites/${rack.spec.siteId}`} className="hover:underline">
                        {rack.spec.siteName || rack.spec.siteId}
                      </Link>
                    ) : "-"}
                  </TableCell>
                  <TableCell>
                    {rack.spec.regionId ? (
                      <Link to={`/infra/regions/${rack.spec.regionId}`} className="hover:underline">
                        {rack.spec.regionName || rack.spec.regionId}
                      </Link>
                    ) : "-"}
                  </TableCell>
                  <TableCell className="text-center">{rack.spec.uHeight ?? 0}</TableCell>
                  <TableCell>{rack.spec.position || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={rack.spec.status === "active" ? "default" : "secondary"}>
                      {rack.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(rack.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {hasPermission(`${permPrefix}:update`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(rack)} title={t("common.edit")}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {hasPermission(`${permPrefix}:delete`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(rack)} title={t("common.delete")}>
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

      <RackFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
        allLocations={allLocations}
      />

      <RackFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        rack={editTarget ?? undefined}
        onSuccess={fetchData}
        allLocations={allLocations}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("rack.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("rack.deleteSelected")}
        description={t("rack.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Rack Form Dialog =====

export interface RackFormValues {
  name: string
  displayName: string
  description: string
  locationId: string
  status: "active" | "inactive"
  uHeight: number | ""
  position: string
  powerCapacity: string
}

export function RackFormDialog({
  open, onOpenChange, rack, onSuccess, allLocations,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  rack?: Rack
  onSuccess: () => void
  allLocations: Location[]
}) {
  const { t } = useTranslation()
  const isEdit = !!rack
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
    displayName: z.string().min(1, t("api.validation.required", { field: t("rack.displayName") })),
    description: z.string().optional(),
    locationId: z.string().min(1, t("api.validation.required", { field: t("rack.locationId") })),
    status: z.enum(["active", "inactive"]),
    uHeight: z.union([z.coerce.number().int().min(0), z.literal("")]).optional().transform(v => v === "" ? undefined : v),
    position: z.string().optional(),
    powerCapacity: z.string().optional(),
  })

  const form = useForm<RackFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      name: "", displayName: "", description: "", locationId: "", status: "active",
      uHeight: "", position: "", powerCapacity: "",
    },
  })

  useEffect(() => {
    if (open) {
      if (rack) {
        form.reset({
          name: rack.metadata.name,
          displayName: rack.spec.displayName ?? "",
          description: rack.spec.description ?? "",
          locationId: rack.spec.locationId ?? "",
          status: (rack.spec.status as "active" | "inactive") ?? "active",
          uHeight: rack.spec.uHeight ?? "",
          position: rack.spec.position ?? "",
          powerCapacity: rack.spec.powerCapacity ?? "",
        })
      } else {
        form.reset({
          name: "", displayName: "", description: "", locationId: "", status: "active",
          uHeight: "", position: "", powerCapacity: "",
        })
      }
    }
  }, [open, rack, form])

  const onSubmit = async (values: RackFormValues) => {
    setLoading(true)
    try {
      const uh = values.uHeight === "" ? undefined : values.uHeight

      if (isEdit) {
        await updateRack(rack.metadata.id, {
          metadata: rack.metadata,
          spec: {
            displayName: values.displayName,
            description: values.description,
            locationId: values.locationId,
            status: values.status,
            uHeight: uh,
            position: values.position,
            powerCapacity: values.powerCapacity,
          } as Rack["spec"],
        })
        toast.success(t("action.updateSuccess"))
      } else {
        await createRack({
          metadata: { name: values.name } as Rack["metadata"],
          spec: {
            displayName: values.displayName,
            description: values.description,
            locationId: values.locationId,
            status: values.status,
            uHeight: uh,
            position: values.position,
            powerCapacity: values.powerCapacity,
          } as Rack["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      handleFormApiError(err, form, t, "rack", "rack.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("rack.edit") : t("rack.create")}</DialogTitle>
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
                  <FormLabel>{t("rack.name")}</FormLabel>
                  <FormControl>
                    <Input {...field} disabled={isEdit} placeholder="my-rack" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="displayName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("rack.displayName")}</FormLabel>
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
                  <FormLabel>{t("rack.description")}</FormLabel>
                  <FormControl><Textarea rows={3} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="locationId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("rack.locationId")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue placeholder={t("rack.selectLocation")} /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {allLocations.map((l) => (
                        <SelectItem key={l.metadata.id} value={l.metadata.id}>
                          {l.spec.displayName || l.metadata.name}
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
              name="uHeight"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("rack.uHeight")}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={0}
                      placeholder="0"
                      value={field.value}
                      onChange={(e) => field.onChange(e.target.value === "" ? "" : Number(e.target.value))}
                      onBlur={field.onBlur}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="position"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("rack.position")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="powerCapacity"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("rack.powerCapacity")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter className="mt-6 pt-4 border-t">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
