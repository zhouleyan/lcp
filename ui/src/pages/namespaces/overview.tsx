import { useNavigate, useOutletContext } from "react-router"
import { Users } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useTranslation } from "@/i18n"
import type { Namespace } from "@/api/types"

export default function NamespaceOverviewPage() {
  const { namespace } = useOutletContext<{ namespace: Namespace }>()
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <div className="space-y-4">
      {/* stats cards */}
      <div className="grid grid-cols-2 gap-4">
        <Card
          className="cursor-pointer transition-colors hover:bg-muted/50"
          onClick={() => navigate("users")}
        >
          <CardContent className="flex items-center gap-4 p-4">
            <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
              <Users className="text-primary h-5 w-5" />
            </div>
            <div>
              <p className="text-2xl font-bold">{namespace.spec.memberCount ?? 0}<span className="text-muted-foreground text-base font-normal">/{namespace.spec.maxMembers || "∞"}</span></p>
              <p className="text-muted-foreground text-sm">{t("namespace.members")}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* details card */}
      <Card>
        <CardHeader>
          <CardTitle>{t("namespace.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("common.name")}</span>
              <p className="font-medium">{namespace.metadata.name}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{namespace.spec.displayName || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("namespace.owner")}</span>
              <p className="font-medium">{namespace.spec.ownerName || namespace.spec.ownerId}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("namespace.workspaceName")}</span>
              <p className="font-medium">{namespace.spec.workspaceName || namespace.spec.workspaceId}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("namespace.visibility")}</span>
              <p>
                <Badge variant={namespace.spec.visibility === "public" ? "default" : "secondary"}>
                  {namespace.spec.visibility === "public" ? t("namespace.visibility.public") : t("namespace.visibility.private")}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.status")}</span>
              <p>
                <Badge variant={namespace.spec.status === "active" ? "default" : "secondary"}>
                  {namespace.spec.status === "active" ? t("common.active") : t("common.inactive")}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("namespace.maxMembers")}</span>
              <p className="font-medium">{namespace.spec.maxMembers || "∞"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("namespace.memberCount")}</span>
              <p className="font-medium">{namespace.spec.memberCount ?? 0}/{namespace.spec.maxMembers || "∞"}</p>
            </div>
            <div className="col-span-2">
              <span className="text-muted-foreground">{t("common.description")}</span>
              <p className="font-medium">{namespace.spec.description || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(namespace.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(namespace.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
