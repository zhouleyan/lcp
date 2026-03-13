import { useCallback, useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router"
import {
  Building2,
  FolderKanban,
  Users,
  Shield,
} from "lucide-react"
import { OverviewCard } from "@/components/overview-card"
import { useTranslation } from "@/i18n"
import type { OverviewSpec } from "@/api/types"
import { getPlatformOverview, getWorkspaceOverview, getNamespaceOverview } from "@/api/dashboard/overview"

interface StatCard {
  labelKey: string
  icon: React.ComponentType<{ className?: string }>
  value: number | null
  to: string
}

// ===== Platform Overview =====

export function PlatformOverviewPage() {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [spec, setSpec] = useState<OverviewSpec | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    try {
      const data = await getPlatformOverview()
      setSpec(data.spec)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  const cards: StatCard[] = [
    { labelKey: "nav.workspaces", icon: Building2, value: spec?.workspaceCount ?? null, to: "/iam/workspaces" },
    { labelKey: "nav.namespaces", icon: FolderKanban, value: spec?.namespaceCount ?? null, to: "/iam/namespaces" },
    { labelKey: "nav.users", icon: Users, value: spec?.userCount ?? null, to: "/iam/users" },
    { labelKey: "nav.roles", icon: Shield, value: spec?.roleCount ?? null, to: "/iam/roles" },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.platform.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.platform.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {cards.map((card) => (
          <OverviewCard key={card.to} label={t(card.labelKey)} icon={card.icon} value={card.value} loading={loading} onClick={() => navigate(card.to)} />
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
  const [spec, setSpec] = useState<OverviewSpec | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    if (!workspaceId) return
    try {
      const data = await getWorkspaceOverview(workspaceId)
      setSpec(data.spec)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId])

  useEffect(() => { fetchData() }, [fetchData])

  const prefix = `/iam/workspaces/${workspaceId}`
  const cards: StatCard[] = [
    { labelKey: "nav.namespaces", icon: FolderKanban, value: spec?.namespaceCount ?? null, to: `${prefix}/namespaces` },
    { labelKey: "nav.users", icon: Users, value: spec?.memberCount ?? null, to: `${prefix}/users` },
    { labelKey: "nav.roles", icon: Shield, value: spec?.roleCount ?? null, to: `${prefix}/roles` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.workspace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.workspace.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
        {cards.map((card) => (
          <OverviewCard key={card.to} label={t(card.labelKey)} icon={card.icon} value={card.value} loading={loading} onClick={() => navigate(card.to)} />
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
  const [spec, setSpec] = useState<OverviewSpec | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    if (!workspaceId || !namespaceId) return
    try {
      const data = await getNamespaceOverview(workspaceId, namespaceId)
      setSpec(data.spec)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId, namespaceId])

  useEffect(() => { fetchData() }, [fetchData])

  const prefix = `/iam/workspaces/${workspaceId}/namespaces/${namespaceId}`
  const cards: StatCard[] = [
    { labelKey: "nav.users", icon: Users, value: spec?.memberCount ?? null, to: `${prefix}/users` },
    { labelKey: "nav.roles", icon: Shield, value: spec?.roleCount ?? null, to: `${prefix}/roles` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.namespace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.namespace.desc")}</p>
      </div>
      <div className="grid grid-cols-2 gap-4">
        {cards.map((card) => (
          <OverviewCard key={card.to} label={t(card.labelKey)} icon={card.icon} value={card.value} loading={loading} onClick={() => navigate(card.to)} />
        ))}
      </div>
    </div>
  )
}
