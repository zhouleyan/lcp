import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getLocation, deleteLocation } from "@/api/infra/locations"
import { listSites } from "@/api/infra/sites"
import { showApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Location, Site, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { LocationFormDialog } from "./list"

export default function LocationDetailPage() {
  const { locationId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [location, setLocation] = useState<Location | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [allSites, setAllSites] = useState<Site[]>([])

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

  useEffect(() => {
    if (editOpen) {
      const params: ListParams = { page: 1, pageSize: SELECT_PAGE_SIZE }
      listSites(params).then(data => setAllSites(data.items ?? [])).catch(() => {})
    }
  }, [editOpen])

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
                <span className="text-muted-foreground">{t("location.rackCount")}</span>
                <p className="font-medium">
                  {(location.spec.rackCount ?? 0) > 0 ? (
                    <Link to={`/infra/racks?locationId=${location.metadata.id}`} className="hover:underline">
                      {location.spec.rackCount}
                    </Link>
                  ) : 0}
                </p>
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
      <LocationFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        location={location}
        onSuccess={fetchLocation}
        allSites={allSites}
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

