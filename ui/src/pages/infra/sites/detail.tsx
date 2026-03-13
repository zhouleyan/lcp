import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2, Warehouse } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getSite, deleteSite, getSiteLocations } from "@/api/infra/sites"
import { listRegions } from "@/api/infra/regions"
import { showApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Site, Region, Location, ListParams } from "@/api/types"
import { OverviewCard } from "@/components/overview-card"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { Pagination } from "@/components/pagination"
import { SiteFormDialog } from "./list"

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

  const [allRegions, setAllRegions] = useState<Region[]>([])

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

  useEffect(() => {
    if (editOpen) {
      listRegions({ page: 1, pageSize: SELECT_PAGE_SIZE }).then(data => setAllRegions(data.items ?? [])).catch(() => {})
    }
  }, [editOpen])

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

      <div className="space-y-6">
        {/* Overview cards */}
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
          <OverviewCard label={t("site.locationCount")} icon={Warehouse} value={site.spec.locationCount ?? 0} />
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
      <SiteFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        site={site}
        onSuccess={fetchSite}
        allRegions={allRegions}
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

