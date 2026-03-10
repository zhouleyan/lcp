import { useCallback, useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router"
import {
  Building2,
  FolderKanban,
  Users,
  Shield,
  ShieldCheck,
  ShieldOff,
} from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useTranslation } from "@/i18n"
import type { OverviewSpec } from "@/api/types"
import { ApiError } from "@/api/client"
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
  const [forbidden, setForbidden] = useState(false)

  const fetchData = useCallback(async () => {
    try {
      const data = await getPlatformOverview()
      setSpec(data.spec)
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setForbidden(true)
      }
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
    { labelKey: "nav.rolebindings", icon: ShieldCheck, value: spec?.roleBindingCount ?? null, to: "/iam/rolebindings" },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.platform.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.platform.desc")}</p>
      </div>
      {forbidden ? (
        <ForbiddenHint t={t} />
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-5">
          {cards.map((card) => (
            <OverviewCard key={card.to} card={card} loading={loading} onClick={() => navigate(card.to)} t={t} />
          ))}
        </div>
      )}
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
  const [forbidden, setForbidden] = useState(false)

  const fetchData = useCallback(async () => {
    if (!workspaceId) return
    try {
      const data = await getWorkspaceOverview(workspaceId)
      setSpec(data.spec)
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setForbidden(true)
      }
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
    { labelKey: "nav.rolebindings", icon: ShieldCheck, value: spec?.roleBindingCount ?? null, to: `${prefix}/rolebindings` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.workspace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.workspace.desc")}</p>
      </div>
      {forbidden ? (
        <ForbiddenHint t={t} />
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {cards.map((card) => (
            <OverviewCard key={card.to} card={card} loading={loading} onClick={() => navigate(card.to)} t={t} />
          ))}
        </div>
      )}
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
  const [forbidden, setForbidden] = useState(false)

  const fetchData = useCallback(async () => {
    if (!workspaceId || !namespaceId) return
    try {
      const data = await getNamespaceOverview(workspaceId, namespaceId)
      setSpec(data.spec)
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setForbidden(true)
      }
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
    { labelKey: "nav.rolebindings", icon: ShieldCheck, value: spec?.roleBindingCount ?? null, to: `${prefix}/rolebindings` },
  ]

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("overview.namespace.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("overview.namespace.desc")}</p>
      </div>
      {forbidden ? (
        <ForbiddenHint t={t} />
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
          {cards.map((card) => (
            <OverviewCard key={card.to} card={card} loading={loading} onClick={() => navigate(card.to)} t={t} />
          ))}
        </div>
      )}
    </div>
  )
}

// ===== Forbidden Hint =====

function ForbiddenHint({ t }: { t: (key: string) => string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <ShieldOff className="text-muted-foreground mb-4 h-12 w-12" />
      <p className="text-muted-foreground text-sm">{t("overview.forbidden")}</p>
    </div>
  )
}

// ===== Shared Card =====

function OverviewCard({
  card,
  loading,
  onClick,
  t,
}: {
  card: StatCard
  loading: boolean
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
          {loading ? (
            <Skeleton className="mb-1 h-7 w-12" />
          ) : (
            <p className="text-2xl font-bold">{card.value ?? "-"}</p>
          )}
          <p className="text-muted-foreground text-sm">{t(card.labelKey)}</p>
        </div>
      </CardContent>
    </Card>
  )
}
