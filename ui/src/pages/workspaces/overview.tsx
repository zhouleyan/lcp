import { useNavigate, useOutletContext } from "react-router"
import { FolderKanban, Users } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useTranslation } from "@/i18n"
import type { Workspace } from "@/api/types"

export default function WorkspaceOverviewPage() {
  const { workspace } = useOutletContext<{ workspace: Workspace }>()
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <div className="space-y-4">
      {/* stats cards */}
      <div className="grid grid-cols-2 gap-4">
        <Card
          className="cursor-pointer transition-colors hover:bg-muted/50"
          onClick={() => navigate("namespaces")}
        >
          <CardContent className="flex items-center gap-4 p-4">
            <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
              <FolderKanban className="text-primary h-5 w-5" />
            </div>
            <div>
              <p className="text-2xl font-bold">{workspace.spec.namespaceCount ?? 0}</p>
              <p className="text-muted-foreground text-sm">{t("workspace.namespaces")}</p>
            </div>
          </CardContent>
        </Card>
        <Card
          className="cursor-pointer transition-colors hover:bg-muted/50"
          onClick={() => navigate("users")}
        >
          <CardContent className="flex items-center gap-4 p-4">
            <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
              <Users className="text-primary h-5 w-5" />
            </div>
            <div>
              <p className="text-2xl font-bold">{workspace.spec.memberCount ?? 0}</p>
              <p className="text-muted-foreground text-sm">{t("workspace.members")}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* details card */}
      <Card>
        <CardHeader>
          <CardTitle>{t("workspace.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("common.name")}</span>
              <p className="font-medium">{workspace.metadata.name}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{workspace.spec.displayName || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("workspace.owner")}</span>
              <p className="font-medium">{workspace.spec.ownerName || workspace.spec.ownerId}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.status")}</span>
              <p>
                <Badge variant={workspace.spec.status === "active" ? "default" : "secondary"}>
                  {workspace.spec.status === "active" ? t("common.active") : t("common.inactive")}
                </Badge>
              </p>
            </div>
            <div className="col-span-2">
              <span className="text-muted-foreground">{t("common.description")}</span>
              <p className="font-medium">{workspace.spec.description || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(workspace.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(workspace.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
