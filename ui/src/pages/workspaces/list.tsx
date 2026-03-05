import { useEffect, useState } from "react"
import { Link } from "react-router"
import { Plus, Trash2 } from "lucide-react"
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
import { listWorkspaces } from "@/api/workspaces"
import type { Workspace } from "@/api/types"
import { useTranslation } from "@/i18n"

export default function WorkspaceListPage() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const { t } = useTranslation()

  useEffect(() => {
    listWorkspaces()
      .then((data) => {
        setWorkspaces(data.items ?? [])
        setTotalCount(data.totalCount)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("workspace.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("workspace.manage", { count: totalCount })}
          </p>
        </div>
        <Button asChild>
          <Link to="/workspaces/new">
            <Plus className="mr-2 h-4 w-4" />
            {t("workspace.create")}
          </Link>
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("common.name")}</TableHead>
              <TableHead>{t("common.displayName")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
              <TableHead>{t("common.created")}</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-32" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-20" />
                  </TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : workspaces.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground py-8 text-center">
                  {t("workspace.noData")}
                </TableCell>
              </TableRow>
            ) : (
              workspaces.map((ws) => (
                <TableRow key={ws.metadata.id}>
                  <TableCell>
                    <Link
                      to={`/workspaces/${ws.metadata.id}`}
                      className="font-medium hover:underline"
                    >
                      {ws.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{ws.spec.displayName || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={ws.spec.status === "active" ? "default" : "secondary"}>
                      {ws.spec.status || "active"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {new Date(ws.metadata.createdAt).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon">
                      <Trash2 className="h-4 w-4" />
                    </Button>
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
