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
import { listUsers } from "@/api/users"
import type { User } from "@/api/types"
import { useTranslation } from "@/i18n"

export default function UserListPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const { t } = useTranslation()

  useEffect(() => {
    listUsers()
      .then((data) => {
        setUsers(data.items ?? [])
        setTotalCount(data.totalCount)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("user.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("user.manage", { count: totalCount })}
          </p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          {t("user.create")}
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("user.username")}</TableHead>
              <TableHead>{t("user.email")}</TableHead>
              <TableHead>{t("common.displayName")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
              <TableHead>{t("common.created")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 5 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-24" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground py-8 text-center">
                  {t("user.noData")}
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.metadata.id}>
                  <TableCell className="font-medium">{user.spec.username}</TableCell>
                  <TableCell>{user.spec.email}</TableCell>
                  <TableCell>{user.spec.displayName || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
                      {user.spec.status || "active"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {new Date(user.metadata.createdAt).toLocaleDateString()}
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
