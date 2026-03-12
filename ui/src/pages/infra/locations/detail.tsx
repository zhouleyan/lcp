import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2 } from "lucide-react"
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
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getLocation, updateLocation, deleteLocation } from "@/api/infra/locations"
import { listSites } from "@/api/infra/sites"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Location, Site, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"

export default function LocationDetailPage() {
  const { locationId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [location, setLocation] = useState<Location | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const permPrefix = "infra:locations"

  const fetchLocation = useCallback(async () => {
    if (!locationId) return
    try {
      const data = await getLocation(locationId)
      setLocation(data)
    } catch {
      setLocation(null)
    } finally {
      setLoading(false)
    }
  }, [locationId])

  useEffect(() => { fetchLocation() }, [fetchLocation])

  const handleDelete = async () => {
    if (!location) return
    try {
      await deleteLocation(location.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "location.title")
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

  if (!location) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("location.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{location.metadata.name}</h1>
          <Badge variant={location.spec.status === "active" ? "default" : "secondary"}>
            {location.spec.status === "active" ? t("common.active") : t("common.inactive")}
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
        {/* Basic info card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("location.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{location.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{location.spec.displayName || "-"}</p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{location.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.siteName")}</span>
                <p className="font-medium">
                  {location.spec.siteId ? (
                    <Link to={`/infra/sites/${location.spec.siteId}`} className="hover:underline">
                      {location.spec.siteName || location.spec.siteId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.regionName")}</span>
                <p className="font-medium">
                  {location.spec.regionId ? (
                    <Link to={`/infra/regions/${location.spec.regionId}`} className="hover:underline">
                      {location.spec.regionName || location.spec.regionId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={location.spec.status === "active" ? "default" : "secondary"}>
                    {location.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.floor")}</span>
                <p className="font-medium">{location.spec.floor || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.rackCapacity")}</span>
                <p className="font-medium">{location.spec.rackCapacity ?? 0}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(location.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(location.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Contact info card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("location.contactInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("location.contactName")}</span>
                <p className="font-medium">{location.spec.contactName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.contactPhone")}</span>
                <p className="font-medium">{location.spec.contactPhone || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("location.contactEmail")}</span>
                <p className="font-medium">{location.spec.contactEmail || "-"}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <EditLocationDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        location={location}
        onSuccess={fetchLocation}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("location.deleteConfirm", { name: location.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Edit Location Dialog =====

function EditLocationDialog({
  open, onOpenChange, location, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  location: Location
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [allSites, setAllSites] = useState<Site[]>([])

  // Fetch sites for select
  useEffect(() => {
    if (open) {
      const params: ListParams = { page: 1, pageSize: 200 }
      listSites(params).then(data => setAllSites(data.items ?? [])).catch(() => {})
    }
  }, [open])

  const schema = z.object({
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

  type FormValues = {
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

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: location.spec.displayName ?? "",
      description: location.spec.description ?? "",
      siteId: location.spec.siteId ?? "",
      status: (location.spec.status as "active" | "inactive") ?? "active",
      floor: location.spec.floor ?? "",
      rackCapacity: location.spec.rackCapacity ?? "",
      contactName: location.spec.contactName ?? "",
      contactPhone: location.spec.contactPhone ?? "",
      contactEmail: location.spec.contactEmail ?? "",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
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
    }
  }, [open, location, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const rc = values.rackCapacity === "" ? undefined : values.rackCapacity
      const payload = {
        metadata: location.metadata,
        spec: {
          ...location.spec,
          displayName: values.displayName,
          description: values.description,
          siteId: values.siteId,
          status: values.status,
          floor: values.floor,
          rackCapacity: rc,
          contactName: values.contactName,
          contactPhone: values.contactPhone,
          contactEmail: values.contactEmail,
        },
      }
      await updateLocation(location.metadata.id, payload)
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const field = d.field.replace(/^(metadata|spec)\./, "") as keyof FormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`location.${field}`) || field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("location.title") }) : err.message })
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
          <DialogTitle>{t("location.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("location.name")}</label>
              <Input value={location.metadata.name} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("location.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("location.description")}</FormLabel><FormControl><Textarea rows={3} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="siteId" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("location.siteId")}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange} disabled>
                  <FormControl><SelectTrigger className="w-full"><SelectValue placeholder={t("location.selectSite")} /></SelectTrigger></FormControl>
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
            <FormField control={form.control} name="floor" render={({ field }) => (
              <FormItem><FormLabel>{t("location.floor")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="rackCapacity" render={({ field }) => (
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
            )} />
            <FormField control={form.control} name="contactName" render={({ field }) => (
              <FormItem><FormLabel>{t("location.contactName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="contactPhone" render={({ field }) => (
              <FormItem><FormLabel>{t("location.contactPhone")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="contactEmail" render={({ field }) => (
              <FormItem><FormLabel>{t("location.contactEmail")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
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
