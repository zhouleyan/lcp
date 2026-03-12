import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router"
import { Pencil, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getRack, deleteRack } from "@/api/infra/racks"
import { listLocations } from "@/api/infra/locations"
import { showApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Rack, Location, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { RackFormDialog } from "./list"

export default function RackDetailPage() {
  const { rackId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [rack, setRack] = useState<Rack | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [allLocations, setAllLocations] = useState<Location[]>([])

  const permPrefix = "infra:racks"

  const fetchRack = useCallback(async () => {
    if (!rackId) return
    try {
      const data = await getRack(rackId)
      setRack(data)
    } catch {
      setRack(null)
    } finally {
      setLoading(false)
    }
  }, [rackId])

  useEffect(() => { fetchRack() }, [fetchRack])

  useEffect(() => {
    if (editOpen) {
      const params: ListParams = { page: 1, pageSize: SELECT_PAGE_SIZE }
      listLocations(params).then(data => setAllLocations(data.items ?? [])).catch(() => {})
    }
  }, [editOpen])

  const handleDelete = async () => {
    if (!rack) return
    try {
      await deleteRack(rack.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "rack.title")
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

  if (!rack) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("rack.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{rack.metadata.name}</h1>
          <Badge variant={rack.spec.status === "active" ? "default" : "secondary"}>
            {rack.spec.status === "active" ? t("common.active") : t("common.inactive")}
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
            <CardTitle>{t("rack.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{rack.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{rack.spec.displayName || "-"}</p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{rack.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.locationName")}</span>
                <p className="font-medium">
                  {rack.spec.locationId ? (
                    <Link to={`/infra/locations/${rack.spec.locationId}`} className="hover:underline">
                      {rack.spec.locationName || rack.spec.locationId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.siteName")}</span>
                <p className="font-medium">
                  {rack.spec.siteId ? (
                    <Link to={`/infra/sites/${rack.spec.siteId}`} className="hover:underline">
                      {rack.spec.siteName || rack.spec.siteId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.regionName")}</span>
                <p className="font-medium">
                  {rack.spec.regionId ? (
                    <Link to={`/infra/regions/${rack.spec.regionId}`} className="hover:underline">
                      {rack.spec.regionName || rack.spec.regionId}
                    </Link>
                  ) : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={rack.spec.status === "active" ? "default" : "secondary"}>
                    {rack.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.uHeight")}</span>
                <p className="font-medium">{rack.spec.uHeight ?? 0}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.position")}</span>
                <p className="font-medium">{rack.spec.position || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("rack.powerCapacity")}</span>
                <p className="font-medium">{rack.spec.powerCapacity || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(rack.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(rack.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <RackFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        rack={rack}
        onSuccess={fetchRack}
        allLocations={allLocations}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("rack.deleteConfirm", { name: rack.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}
