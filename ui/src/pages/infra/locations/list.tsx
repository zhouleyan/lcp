import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router"
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
  listLocations, createLocation, updateLocation, deleteLocation, deleteLocations,
} from "@/api/infra/locations"
import { listRegions } from "@/api/infra/regions"
import { listSites } from "@/api/infra/sites"
import { showApiError, handleFormApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Location, Region, Site, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

export default function LocationListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const [locations, setLocations] = useState<Location[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")
  const [regionFilter, setRegionFilter] = useState("all")
  const [siteFilter, setSiteFilter] = useState("all")
  const [allRegions, setAllRegions] = useState<Region[]>([])
  const [allSites, setAllSites] = useState<Site[]>([])

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Location | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Location | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const permPrefix = "infra:locations"

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

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      if (siteFilter !== "all") params.siteId = siteFilter
      if (regionFilter !== "all") params.regionId = regionFilter
      const data = await listLocations(params)
      setLocations(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter, siteFilter, regionFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, siteFilter, regionFilter, pageSize])
  useEffect(() => { clearSelection() }, [locations])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteLocation(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "location.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteLocations(ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "location.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("location.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("location.manage", { count: totalCount })}
          </p>
        </div>
        {hasPermission(`${permPrefix}:create`) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("location.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("location.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("location.deleteSelected")} ({selected.size})
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
                    checked={locations.length > 0 && selected.size === locations.length}
                    onCheckedChange={() => toggleAll(locations.map((l) => l.metadata.id))}
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
                      {t("location.siteName")}
                      <Filter className={`h-3 w-3 ${siteFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setSiteFilter("all")}>{t("location.filter.siteAll")}</DropdownMenuItem>
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
                      {t("location.regionName")}
                      <Filter className={`h-3 w-3 ${regionFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setRegionFilter("all")}>{t("location.filter.regionAll")}</DropdownMenuItem>
                    {allRegions.map((r) => (
                      <DropdownMenuItem key={r.metadata.id} onClick={() => setRegionFilter(r.metadata.id)}>
                        {r.spec.displayName || r.metadata.name}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>{t("location.floor")}</TableHead>
              <TableHead className="text-center">{t("location.rackCapacity")}</TableHead>
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
                  {Array.from({ length: 10 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : locations.length === 0 ? (
              <TableRow>
                <TableCell colSpan={10} className="text-muted-foreground py-8 text-center">
                  {t("location.noData")}
                </TableCell>
              </TableRow>
            ) : (
              locations.map((loc) => (
                <TableRow key={loc.metadata.id}>
                  <TableCell>
                    {hasPermission(`${permPrefix}:deleteCollection`) && (
                      <Checkbox
                        checked={selected.has(loc.metadata.id)}
                        onCheckedChange={() => toggleOne(loc.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Link to={`${loc.metadata.id}`} className="font-medium hover:underline">
                      {loc.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{loc.spec.displayName || "-"}</TableCell>
                  <TableCell>
                    {loc.spec.siteId ? (
                      <Link to={`/infra/sites/${loc.spec.siteId}`} className="hover:underline">
                        {loc.spec.siteName || loc.spec.siteId}
                      </Link>
                    ) : "-"}
                  </TableCell>
                  <TableCell>
                    {loc.spec.regionId ? (
                      <Link to={`/infra/regions/${loc.spec.regionId}`} className="hover:underline">
                        {loc.spec.regionName || loc.spec.regionId}
                      </Link>
                    ) : "-"}
                  </TableCell>
                  <TableCell>{loc.spec.floor || "-"}</TableCell>
                  <TableCell className="text-center">{loc.spec.rackCapacity ?? 0}</TableCell>
                  <TableCell>
                    <Badge variant={loc.spec.status === "active" ? "default" : "secondary"}>
                      {loc.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(loc.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {hasPermission(`${permPrefix}:update`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(loc)} title={t("common.edit")}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {hasPermission(`${permPrefix}:delete`) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(loc)} title={t("common.delete")}>
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

      <LocationFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
        allSites={allSites}
      />

      <LocationFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        location={editTarget ?? undefined}
        onSuccess={fetchData}
        allSites={allSites}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("location.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("location.deleteSelected")}
        description={t("location.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Location Form Dialog =====

export interface LocationFormValues {
  name: string
  displayName: string
  description: string
  siteId: string
  status: "active" | "inactive"
  floor: string
  rackCapacity: number | ""
  contactName: string
  contactPhone: string
  contactEmail: string
}

export function LocationFormDialog({
  open, onOpenChange, location, onSuccess, allSites,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  location?: Location
  onSuccess: () => void
  allSites: Site[]
}) {
  const { t } = useTranslation()
  const isEdit = !!location
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
    displayName: z.string().min(1, t("api.validation.required", { field: t("location.displayName") })),
    description: z.string().optional(),
    siteId: z.string().min(1, t("api.validation.required", { field: t("location.siteId") })),
    status: z.enum(["active", "inactive"]),
    floor: z.string().optional(),
    rackCapacity: z.union([z.coerce.number().int().min(0), z.literal("")]).optional().transform(v => v === "" ? undefined : v),
    contactName: z.string().optional(),
    contactPhone: z.string().optional(),
    contactEmail: z.string().optional(),
  })

  const form = useForm<LocationFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      name: "", displayName: "", description: "", siteId: "", status: "active",
      floor: "", rackCapacity: "", contactName: "", contactPhone: "", contactEmail: "",
    },
  })

  useEffect(() => {
    if (open) {
      if (location) {
        form.reset({
          name: location.metadata.name,
          displayName: location.spec.displayName ?? "",
          description: location.spec.description ?? "",
          siteId: location.spec.siteId ?? "",
          status: (location.spec.status as "active" | "inactive") ?? "active",
          floor: location.spec.floor ?? "",
          rackCapacity: location.spec.rackCapacity ?? "",
          contactName: location.spec.contactName ?? "",
          contactPhone: location.spec.contactPhone ?? "",
          contactEmail: location.spec.contactEmail ?? "",
        })
      } else {
        form.reset({
          name: "", displayName: "", description: "", siteId: "", status: "active",
          floor: "", rackCapacity: "", contactName: "", contactPhone: "", contactEmail: "",
        })
      }
    }
  }, [open, location, form])

  const onSubmit = async (values: LocationFormValues) => {
    setLoading(true)
    try {
      const rc = values.rackCapacity === "" ? undefined : values.rackCapacity

      if (isEdit) {
        await updateLocation(location.metadata.id, {
          metadata: location.metadata,
          spec: {
            displayName: values.displayName,
            description: values.description,
            siteId: values.siteId,
            status: values.status,
            floor: values.floor,
            rackCapacity: rc,
            contactName: values.contactName,
            contactPhone: values.contactPhone,
            contactEmail: values.contactEmail,
          } as Location["spec"],
        })
        toast.success(t("action.updateSuccess"))
      } else {
        await createLocation({
          metadata: { name: values.name } as Location["metadata"],
          spec: {
            displayName: values.displayName,
            description: values.description,
            siteId: values.siteId,
            status: values.status,
            floor: values.floor,
            rackCapacity: rc,
            contactName: values.contactName,
            contactPhone: values.contactPhone,
            contactEmail: values.contactEmail,
          } as Location["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      handleFormApiError(err, form, t, "location", "location.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("location.edit") : t("location.create")}</DialogTitle>
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
                  <FormLabel>{t("location.name")}</FormLabel>
                  <FormControl>
                    <Input {...field} disabled={isEdit} placeholder="my-location" />
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
                  <FormLabel>{t("location.displayName")}</FormLabel>
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
                  <FormLabel>{t("location.description")}</FormLabel>
                  <FormControl><Textarea rows={3} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="siteId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.siteId")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue placeholder={t("location.selectSite")} /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {allSites.map((s) => (
                        <SelectItem key={s.metadata.id} value={s.metadata.id}>
                          {s.spec.displayName || s.metadata.name}
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
              name="floor"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.floor")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="rackCapacity"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.rackCapacity")}</FormLabel>
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
              name="contactName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.contactName")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="contactPhone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.contactPhone")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="contactEmail"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("location.contactEmail")}</FormLabel>
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
