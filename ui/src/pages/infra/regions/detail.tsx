import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2, MapPin } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getRegion, deleteRegion, getRegionSites } from "@/api/infra/regions"
import { showApiError } from "@/api/client"
import type { Region, Site, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { Pagination } from "@/components/pagination"
import { RegionFormDialog } from "./list"

export default function RegionDetailPage() {
  const { regionId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [region, setRegion] = useState<Region | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Sites in this region
  const [sites, setSites] = useState<Site[]>([])
  const [sitesLoading, setSitesLoading] = useState(true)
  const [sitesTotal, setSitesTotal] = useState(0)
  const [sitesPage, setSitesPage] = useState(1)
  const [sitesPageSize, setSitesPageSize] = useState(10)

  const permPrefix = "infra:regions"

  const fetchRegion = useCallback(async () => {
    if (!regionId) return
    try {
      const data = await getRegion(regionId)
      setRegion(data)
    } catch {
      setRegion(null)
    } finally {
      setLoading(false)
    }
  }, [regionId])

  const fetchSites = useCallback(async () => {
    if (!regionId) return
    setSitesLoading(true)
    try {
      const params: ListParams = { page: sitesPage, pageSize: sitesPageSize }
      const data = await getRegionSites(regionId, params)
      setSites(data.items ?? [])
      setSitesTotal(data.totalCount)
    } catch {
      setSites([])
    } finally {
      setSitesLoading(false)
    }
  }, [regionId, sitesPage, sitesPageSize])

  useEffect(() => { fetchRegion() }, [fetchRegion])
  useEffect(() => { fetchSites() }, [fetchSites])

  const handleDelete = async () => {
    if (!region) return
    try {
      await deleteRegion(region.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "region.title")
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

  if (!region) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("region.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{region.metadata.name}</h1>
          <Badge variant={region.spec.status === "active" ? "default" : "secondary"}>
            {region.spec.status === "active" ? t("common.active") : t("common.inactive")}
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
                <MapPin className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{region.spec.siteCount ?? 0}</p>
                <p className="text-muted-foreground text-sm">{t("region.siteCount")}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Details card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("region.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{region.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{region.spec.displayName || "-"}</p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{region.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={region.spec.status === "active" ? "default" : "secondary"}>
                    {region.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("region.latitude")}</span>
                <p className="font-medium">{region.spec.latitude ?? "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("region.longitude")}</span>
                <p className="font-medium">{region.spec.longitude ?? "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(region.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(region.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Sites in this region */}
        <Card>
          <CardHeader>
            <CardTitle>{t("region.sites")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("common.displayName")}</TableHead>
                    <TableHead>{t("common.status")}</TableHead>
                    <TableHead className="text-center">{t("site.locationCount")}</TableHead>
                    <TableHead>{t("common.created")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sitesLoading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                      <TableRow key={i}>
                        {Array.from({ length: 5 }).map((_, j) => (
                          <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : sites.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-muted-foreground py-8 text-center">
                        {t("region.sitesEmpty")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    sites.map((site) => (
                      <TableRow key={site.metadata.id}>
                        <TableCell>
                          <Link to={`/infra/sites/${site.metadata.id}`} className="font-medium hover:underline">
                            {site.metadata.name}
                          </Link>
                        </TableCell>
                        <TableCell>{site.spec.displayName || "-"}</TableCell>
                        <TableCell>
                          <Badge variant={site.spec.status === "active" ? "default" : "secondary"}>
                            {site.spec.status === "active" ? t("common.active") : t("common.inactive")}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-center">{site.spec.locationCount ?? 0}</TableCell>
                        <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                          {new Date(site.metadata.createdAt).toLocaleString()}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
            <Pagination
              totalCount={sitesTotal}
              page={sitesPage}
              pageSize={sitesPageSize}
              onPageChange={setSitesPage}
              onPageSizeChange={setSitesPageSize}
            />
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <RegionFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        region={region}
        onSuccess={fetchRegion}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("region.deleteConfirm", { name: region.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

