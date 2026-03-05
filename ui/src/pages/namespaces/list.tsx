import { useEffect, useState } from "react"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { listNamespaces } from "@/api/namespaces"
import type { Namespace } from "@/api/types"
import { useTranslation } from "@/i18n"

export default function NamespaceListPage() {
  const [namespaces, setNamespaces] = useState<Namespace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const { t } = useTranslation()

  useEffect(() => {
    listNamespaces()
      .then((data) => {
        setNamespaces(data.items ?? [])
        setTotalCount(data.totalCount)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("namespace.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("namespace.manage", { count: totalCount })}
          </p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          {t("namespace.create")}
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("common.name")}</TableHead>
              <TableHead>{t("common.displayName")}</TableHead>
              <TableHead>{t("namespace.workspaceId")}</TableHead>
              <TableHead>{t("namespace.visibility")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
              <TableHead>{t("common.created")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 6 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-24" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : namespaces.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-muted-foreground py-8 text-center">
                  {t("namespace.noData")}
                </TableCell>
              </TableRow>
            ) : (
              namespaces.map((ns) => (
                <TableRow key={ns.metadata.id}>
                  <TableCell className="font-medium">{ns.metadata.name}</TableCell>
                  <TableCell>{ns.spec.displayName || "-"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {ns.spec.workspaceId}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{ns.spec.visibility || "public"}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={ns.spec.status === "active" ? "default" : "secondary"}>
                      {ns.spec.status || "active"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {new Date(ns.metadata.createdAt).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
