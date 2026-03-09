import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { getWorkspaceUser, getNamespaceUser } from "@/api/users"
import type { User } from "@/api/types"
import { useTranslation } from "@/i18n"

export default function ScopedUserDetailPage() {
  const { workspaceId, namespaceId, userId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  // Build base path for back navigation
  const basePath = namespaceId
    ? `/workspaces/${workspaceId}/namespaces/${namespaceId}/users`
    : `/workspaces/${workspaceId}/users`

  const fetchUser = useCallback(async () => {
    if (!userId) return
    try {
      const u = namespaceId
        ? await getNamespaceUser(workspaceId!, namespaceId, userId)
        : await getWorkspaceUser(workspaceId!, userId)
      setUser(u)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [userId, workspaceId, namespaceId])

  useEffect(() => { fetchUser() }, [fetchUser])

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!user) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("user.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center gap-3">
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate(basePath)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold">{user.spec.username}</h1>
        <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
          {user.spec.status === "active" ? t("common.active") : t("common.inactive")}
        </Badge>
      </div>

      {/* user info card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>{t("user.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("user.username")}</span>
              <p className="font-medium">{user.spec.username}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{user.spec.displayName || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("user.email")}</span>
              <p className="font-medium">{user.spec.email}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.phone")}</span>
              <p className="font-medium">{user.spec.phone || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.status")}</span>
              <p>
                <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
                  {user.spec.status === "active" ? t("common.active") : t("common.inactive")}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("user.role")}</span>
              <p className="font-medium">
                {user.spec.role
                  ? <Badge variant="outline">{t(`role.${user.spec.role}`, { defaultValue: user.spec.role })}</Badge>
                  : "-"}
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(user.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(user.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
