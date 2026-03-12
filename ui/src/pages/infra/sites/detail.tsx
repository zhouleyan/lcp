import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2, Warehouse } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getSite, updateSite, deleteSite, getSiteLocations } from "@/api/infra/sites"
import { listRegions } from "@/api/infra/regions"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Site, Region, Location, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { Pagination } from "@/components/pagination"

export default function SiteDetailPage() {
  const { siteId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [site, setSite] = useState<Site | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Locations in this site
  const [locations, setLocations] = useState<Location[]>([])
  const [locationsLoading, setLocationsLoading] = useState(true)
  const [locationsTotal, setLocationsTotal] = useState(0)
  const [locationsPage, setLocationsPage] = useState(1)
  const [locationsPageSize, setLocationsPageSize] = useState(10)

  const permPrefix = "infra:sites"

  const fetchSite = useCallback(async () => {
    if (!siteId) return
    try {
      const data = await getSite(siteId)
      setSite(data)
    } catch {
      setSite(null)
    } finally {
      setLoading(false)
    }
  }, [siteId])

  const fetchLocations = useCallback(async () => {
    if (!siteId) return
    setLocationsLoading(true)
    try {
      const params: ListParams = { page: locationsPage, pageSize: locationsPageSize }
      const data = await getSiteLocations(siteId, params)
      setLocations(data.items ?? [])
      setLocationsTotal(data.totalCount)
    } catch {
      setLocations([])
    } finally {
      setLocationsLoading(false)
    }
  }, [siteId, locationsPage, locationsPageSize])

  useEffect(() => { fetchSite() }, [fetchSite])
  useEffect(() => { fetchLocations() }, [fetchLocations])

  const handleDelete = async () => {
    if (!site) return
    try {
      await deleteSite(site.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "site.title")
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!site) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("site.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{site.metadata.name}</h1>
          <Badge variant={site.spec.status === "active" ? "default" : "secondary"}>
            {site.spec.status === "active" ? t("common.active") : t("common.inactive")}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission(`${permPrefix}:update`) && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission(`${permPrefix}:delete`) && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {/* Overview cards */}
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
                <Warehouse className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{site.spec.locationCount ?? 0}</p>
                <p className="text-muted-foreground text-sm">{t("site.locationCount")}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Basic info card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("site.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{site.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{site.spec.displayName || "-"}</p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{site.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("site.regionName")}</span>
                <p className="font-medium">
                  {site.spec.regionId ? (
                    <Link to={`/infra/regions/${site.spec.regionId}`} className="hover:underline">
                      {site.spec.regionName || site.spec.regionId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={site.spec.status === "active" ? "default" : "secondary"}>
                    {site.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("site.address")}</span>
                <p className="font-medium">{site.spec.address || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("site.latitude")}</span>
                <p className="font-medium">{site.spec.latitude ?? "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("site.longitude")}</span>
                <p className="font-medium">{site.spec.longitude ?? "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(site.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(site.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Contact info card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("site.contactInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("site.contactName")}</span>
                <p className="font-medium">{site.spec.contactName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("site.contactPhone")}</span>
                <p className="font-medium">{site.spec.contactPhone || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("site.contactEmail")}</span>
                <p className="font-medium">{site.spec.contactEmail || "-"}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Locations in this site */}
        <Card>
          <CardHeader>
            <CardTitle>{t("site.locations")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("common.displayName")}</TableHead>
                    <TableHead>{t("location.floor")}</TableHead>
                    <TableHead className="text-center">{t("location.rackCapacity")}</TableHead>
                    <TableHead>{t("common.status")}</TableHead>
                    <TableHead>{t("common.created")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {locationsLoading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                      <TableRow key={i}>
                        {Array.from({ length: 6 }).map((_, j) => (
                          <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : locations.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-muted-foreground py-8 text-center">
                        {t("site.locationsEmpty")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    locations.map((loc) => (
                      <TableRow key={loc.metadata.id}>
                        <TableCell>
                          <Link to={`/infra/locations/${loc.metadata.id}`} className="font-medium hover:underline">
                            {loc.metadata.name}
                          </Link>
                        </TableCell>
                        <TableCell>{loc.spec.displayName || "-"}</TableCell>
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
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
            <Pagination
              totalCount={locationsTotal}
              page={locationsPage}
              pageSize={locationsPageSize}
              onPageChange={setLocationsPage}
              onPageSizeChange={setLocationsPageSize}
            />
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <EditSiteDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        site={site}
        onSuccess={fetchSite}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("site.deleteConfirm", { name: site.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Edit Site Dialog =====

function EditSiteDialog({
  open, onOpenChange, site, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  site: Site
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [allRegions, setAllRegions] = useState<Region[]>([])

  // Fetch regions for select
  useEffect(() => {
    if (open) {
      listRegions({ page: 1, pageSize: 200 }).then(data => setAllRegions(data.items ?? [])).catch(() => {})
    }
  }, [open])

  const schema = z.object({
    displayName: z.string().min(1, t("api.validation.required", { field: t("site.displayName") })),
    description: z.string().optional(),
    regionId: z.string().min(1, t("api.validation.required", { field: t("site.regionId") })),
    status: z.enum(["active", "inactive"]),
    address: z.string().optional(),
    latitude: z.union([z.coerce.number().min(-90).max(90), z.literal("")]).optional().transform(v => v === "" ? undefined : v),
    longitude: z.union([z.coerce.number().min(-180).max(180), z.literal("")]).optional().transform(v => v === "" ? undefined : v),
    contactName: z.string().optional(),
    contactPhone: z.string().optional(),
    contactEmail: z.string().optional(),
  })

  type FormValues = {
    displayName: string
    description: string
    regionId: string
    status: "active" | "inactive"
    address: string
    latitude: number | ""
    longitude: number | ""
    contactName: string
    contactPhone: string
    contactEmail: string
  }

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: site.spec.displayName ?? "",
      description: site.spec.description ?? "",
      regionId: site.spec.regionId ?? "",
      status: (site.spec.status as "active" | "inactive") ?? "active",
      address: site.spec.address ?? "",
      latitude: site.spec.latitude ?? "",
      longitude: site.spec.longitude ?? "",
      contactName: site.spec.contactName ?? "",
      contactPhone: site.spec.contactPhone ?? "",
      contactEmail: site.spec.contactEmail ?? "",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: site.spec.displayName ?? "",
        description: site.spec.description ?? "",
        regionId: site.spec.regionId ?? "",
        status: (site.spec.status as "active" | "inactive") ?? "active",
        address: site.spec.address ?? "",
        latitude: site.spec.latitude ?? "",
        longitude: site.spec.longitude ?? "",
        contactName: site.spec.contactName ?? "",
        contactPhone: site.spec.contactPhone ?? "",
        contactEmail: site.spec.contactEmail ?? "",
      })
    }
  }, [open, site, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const lat = values.latitude === "" ? undefined : values.latitude
      const lng = values.longitude === "" ? undefined : values.longitude
      const payload = {
        metadata: site.metadata,
        spec: {
          ...site.spec,
          displayName: values.displayName,
          description: values.description,
          regionId: values.regionId,
          status: values.status,
          address: values.address,
          latitude: lat,
          longitude: lng,
          contactName: values.contactName,
          contactPhone: values.contactPhone,
          contactEmail: values.contactEmail,
        },
      }
      await updateSite(site.metadata.id, payload)
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const field = d.field.replace(/^(metadata|spec)\./, "") as keyof FormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`site.${field}`) || field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("site.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("site.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("site.name")}</label>
              <Input value={site.metadata.name} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("site.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("site.description")}</FormLabel><FormControl><Textarea rows={3} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="regionId" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("site.regionId")}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange} disabled>
                  <FormControl><SelectTrigger className="w-full"><SelectValue placeholder={t("site.selectRegion")} /></SelectTrigger></FormControl>
                  <SelectContent>
                    {allRegions.map((r) => (
                      <SelectItem key={r.metadata.id} value={r.metadata.id}>
                        {r.spec.displayName || r.metadata.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="status" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("common.status")}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                  <SelectContent>
                    <SelectItem value="active">{t("common.active")}</SelectItem>
                    <SelectItem value="inactive">{t("common.inactive")}</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="address" render={({ field }) => (
              <FormItem><FormLabel>{t("site.address")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="latitude" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("site.latitude")}</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    step="any"
                    placeholder="-90 ~ 90"
                    value={field.value}
                    onChange={(e) => field.onChange(e.target.value === "" ? "" : Number(e.target.value))}
                    onBlur={field.onBlur}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="longitude" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("site.longitude")}</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    step="any"
                    placeholder="-180 ~ 180"
                    value={field.value}
                    onChange={(e) => field.onChange(e.target.value === "" ? "" : Number(e.target.value))}
                    onBlur={field.onBlur}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="contactName" render={({ field }) => (
              <FormItem><FormLabel>{t("site.contactName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="contactPhone" render={({ field }) => (
              <FormItem><FormLabel>{t("site.contactPhone")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="contactEmail" render={({ field }) => (
              <FormItem><FormLabel>{t("site.contactEmail")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
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
