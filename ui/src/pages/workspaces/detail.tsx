import { useEffect, useState } from "react"
import { useParams, Link } from "react-router"
import { ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { getWorkspace } from "@/api/workspaces"
import type { Workspace } from "@/api/types"
import { useTranslation } from "@/i18n"

export default function WorkspaceDetailPage() {
  const { workspaceId } = useParams()
  const [workspace, setWorkspace] = useState<Workspace | null>(null)
  const [loading, setLoading] = useState(true)
  const { t } = useTranslation()

  useEffect(() => {
    if (!workspaceId) return
    getWorkspace(workspaceId)
      .then(setWorkspace)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [workspaceId])

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!workspace) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("workspace.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/workspaces">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("workspace.backToList")}
          </Link>
        </Button>
      </div>

      <div className="mb-6 flex items-center gap-3">
        <h1 className="text-2xl font-bold">{workspace.metadata.name}</h1>
        <Badge variant={workspace.spec.status === "active" ? "default" : "secondary"}>
          {workspace.spec.status || "active"}
        </Badge>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">{t("workspace.overview")}</TabsTrigger>
          <TabsTrigger value="namespaces">{t("workspace.namespaces")}</TabsTrigger>
          <TabsTrigger value="members">{t("workspace.members")}</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("workspace.details")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">{t("common.displayName")}</span>
                  <p className="font-medium">{workspace.spec.displayName || "-"}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t("workspace.ownerId")}</span>
                  <p className="font-medium">{workspace.spec.ownerId}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t("common.description")}</span>
                  <p className="font-medium">{workspace.spec.description || "-"}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t("common.created")}</span>
                  <p className="font-medium">
                    {new Date(workspace.metadata.createdAt).toLocaleString()}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="namespaces" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("workspace.namespaces")}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">{t("workspace.namespacesComingSoon")}</p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="members" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("workspace.members")}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">{t("workspace.membersComingSoon")}</p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
