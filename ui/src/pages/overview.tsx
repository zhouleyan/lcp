import { useNavigate, useParams } from "react-router"
import {
  Building2,
  FolderKanban,
  Users,
  Shield,
  LayoutDashboard,
} from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { useTranslation } from "@/i18n"

interface StatCard {
  labelKey: string
  icon: React.ComponentType<{ className?: string }>
  value: string
  to: string
}

// ===== Platform Overview =====

export function PlatformOverviewPage() {
  const navigate = useNavigate()
  const { t } = useTranslation()

  const cards: StatCard[] = [
    { labelKey: "nav.workspaces", icon: Building2, value: "-", to: "/iam/workspaces" },
    { labelKey: "nav.namespaces", icon: FolderKanban, value: "-", to: "/iam/namespaces" },
    { labelKey: "nav.users", icon: Users, value: "-", to: "/iam/users" },
    { labelKey: "nav.roles", icon: Shield, value: "-", to: "/iam/roles" },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.platform.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.platform.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {cards.map((card) => (
          <OverviewCard key={card.to} card={card} onClick={() => navigate(card.to)} t={t} />
        ))}
      </div>
    </div>
  )
}

// ===== Workspace Overview =====

export function WorkspaceOverviewPage() {
  const { workspaceId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const prefix = `/iam/workspaces/${workspaceId}`
  const cards: StatCard[] = [
    { labelKey: "nav.namespaces", icon: FolderKanban, value: "-", to: `${prefix}/namespaces` },
    { labelKey: "nav.users", icon: Users, value: "-", to: `${prefix}/users` },
    { labelKey: "nav.roles", icon: Shield, value: "-", to: `${prefix}/roles` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.workspace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.workspace.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
        {cards.map((card) => (
          <OverviewCard key={card.to} card={card} onClick={() => navigate(card.to)} t={t} />
        ))}
      </div>
    </div>
  )
}

// ===== Namespace Overview =====

export function NamespaceOverviewPage() {
  const { workspaceId, namespaceId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const prefix = `/iam/workspaces/${workspaceId}/namespaces/${namespaceId}`
  const cards: StatCard[] = [
    { labelKey: "nav.users", icon: Users, value: "-", to: `${prefix}/users` },
    { labelKey: "nav.roles", icon: Shield, value: "-", to: `${prefix}/roles` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.namespace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.namespace.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4">
        {cards.map((card) => (
          <OverviewCard key={card.to} card={card} onClick={() => navigate(card.to)} t={t} />
        ))}
      </div>
    </div>
  )
}

// ===== Shared Card =====

function OverviewCard({
  card,
  onClick,
  t,
}: {
  card: StatCard
  onClick: () => void
  t: (key: string) => string
}) {
  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-muted/50"
      onClick={onClick}
    >
      <CardContent className="flex items-center gap-4 p-4">
        <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
          <card.icon className="text-primary h-5 w-5" />
        </div>
        <div>
          <p className="text-2xl font-bold">{card.value}</p>
          <p className="text-muted-foreground text-sm">{t(card.labelKey)}</p>
        </div>
      </CardContent>
    </Card>
  )
}
