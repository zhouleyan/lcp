import { useEffect, useLayoutEffect, useMemo, useState } from "react"
import { Link, Navigate, Outlet, useLocation } from "react-router"
import {
  LayoutDashboard,
  Users,
  Building2,
  FolderKanban,
  FileText,
  Shield,
  ShieldCheck,
  Home,
  ScrollText,
  Server,
  Layers,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { TooltipProvider } from "@/components/ui/tooltip"
import { LanguageSwitcher } from "@/components/language-switcher"
import { UserMenu } from "@/components/user-menu"
import { AppBreadcrumb } from "@/components/app-breadcrumb"
import { ScopeSelector } from "@/components/scope-selector"
import { useTranslation } from "@/i18n"
import { isAuthenticated, startAuthFlow } from "@/lib/auth"
import { useAuthStore } from "@/stores/auth-store"
import { usePermissionStore } from "@/stores/permission-store"
import { usePermission, getDefaultPath } from "@/hooks/use-permission"
import { useScopeStore } from "@/stores/scope-store"
import { isModulePrefix } from "@/modules"

interface NavItem {
  to: string
  labelKey: string
  icon: React.ComponentType<{ className?: string }>
  permission?: string
  permissionScope?: { workspaceId?: string; namespaceId?: string }
}

interface NavGroup {
  labelKey?: string
  items: NavItem[]
}

function buildNavGroups(scopeWorkspaceId: string | null, scopeNamespaceId: string | null): NavGroup[] {
  if (scopeWorkspaceId && scopeNamespaceId) {
    const iamPrefix = `/iam/workspaces/${scopeWorkspaceId}/namespaces/${scopeNamespaceId}`
    const dashPrefix = `/dashboard/workspaces/${scopeWorkspaceId}/namespaces/${scopeNamespaceId}`
    const infraPrefix = `/infra/workspaces/${scopeWorkspaceId}/namespaces/${scopeNamespaceId}`
    const nsScope = { workspaceId: scopeWorkspaceId, namespaceId: scopeNamespaceId }
    return [
      { items: [{ to: `${dashPrefix}/overview`, labelKey: "nav.overview", icon: Home, permission: "dashboard:overview:list", permissionScope: nsScope }] },
      {
        labelKey: "nav.iam",
        items: [
          { to: `${iamPrefix}/users`, labelKey: "nav.users", icon: Users, permission: "iam:users:list", permissionScope: nsScope },
          { to: `${iamPrefix}/roles`, labelKey: "nav.roles", icon: Shield, permission: "iam:roles:list", permissionScope: nsScope },
          { to: `${iamPrefix}/rolebindings`, labelKey: "nav.rolebindings", icon: ShieldCheck, permission: "iam:rolebindings:list", permissionScope: nsScope },
        ],
      },
      {
        labelKey: "nav.infra",
        items: [
          { to: `${infraPrefix}/hosts`, labelKey: "nav.hosts", icon: Server, permission: "infra:hosts:list", permissionScope: nsScope },
          { to: `${infraPrefix}/environments`, labelKey: "nav.environments", icon: Layers, permission: "infra:environments:list", permissionScope: nsScope },
        ],
      },
    ]
  }
  if (scopeWorkspaceId) {
    const iamPrefix = `/iam/workspaces/${scopeWorkspaceId}`
    const dashPrefix = `/dashboard/workspaces/${scopeWorkspaceId}`
    const infraPrefix = `/infra/workspaces/${scopeWorkspaceId}`
    const wsScope = { workspaceId: scopeWorkspaceId }
    return [
      { items: [{ to: `${dashPrefix}/overview`, labelKey: "nav.overview", icon: Home, permission: "dashboard:overview:list", permissionScope: wsScope }] },
      {
        labelKey: "nav.iam",
        items: [
          { to: `${iamPrefix}/namespaces`, labelKey: "nav.namespaces", icon: FolderKanban, permission: "iam:namespaces:list", permissionScope: wsScope },
          { to: `${iamPrefix}/users`, labelKey: "nav.users", icon: Users, permission: "iam:users:list", permissionScope: wsScope },
          { to: `${iamPrefix}/roles`, labelKey: "nav.roles", icon: Shield, permission: "iam:roles:list", permissionScope: wsScope },
          { to: `${iamPrefix}/rolebindings`, labelKey: "nav.rolebindings", icon: ShieldCheck, permission: "iam:rolebindings:list", permissionScope: wsScope },
        ],
      },
      {
        labelKey: "nav.infra",
        items: [
          { to: `${infraPrefix}/hosts`, labelKey: "nav.hosts", icon: Server, permission: "infra:hosts:list", permissionScope: wsScope },
          { to: `${infraPrefix}/environments`, labelKey: "nav.environments", icon: Layers, permission: "infra:environments:list", permissionScope: wsScope },
        ],
      },
    ]
  }
  return [
    { items: [{ to: "/dashboard/overview", labelKey: "nav.overview", icon: Home, permission: "dashboard:overview:list" }] },
    {
      labelKey: "nav.iam",
      items: [
        { to: "/iam/workspaces", labelKey: "nav.workspaces", icon: Building2, permission: "iam:workspaces:list" },
        { to: "/iam/namespaces", labelKey: "nav.namespaces", icon: FolderKanban, permission: "iam:namespaces:list" },
        { to: "/iam/users", labelKey: "nav.users", icon: Users, permission: "iam:users:list" },
        { to: "/iam/roles", labelKey: "nav.roles", icon: Shield, permission: "iam:roles:list" },
        { to: "/iam/rolebindings", labelKey: "nav.rolebindings", icon: ShieldCheck, permission: "iam:rolebindings:list" },
      ],
    },
    {
      labelKey: "nav.infra",
      items: [
        { to: "/infra/hosts", labelKey: "nav.hosts", icon: Server, permission: "infra:hosts:list" },
        { to: "/infra/environments", labelKey: "nav.environments", icon: Layers, permission: "infra:environments:list" },
      ],
    },
    {
      labelKey: "nav.audit",
      items: [
        { to: "/audit/logs", labelKey: "nav.auditLogs", icon: ScrollText, permission: "audit:logs:list" },
      ],
    },
  ]
}

export default function RootLayout() {
  const location = useLocation()
  const { t } = useTranslation()
  const fetchUser = useAuthStore((s) => s.fetchUser)
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions)
  const { hasPermission } = usePermission()
  const scopeWorkspaceId = useScopeStore((s) => s.workspaceId)
  const scopeNamespaceId = useScopeStore((s) => s.namespaceId)
  const setScope = useScopeStore((s) => s.setScope)

  // Sync scope store from URL when navigating via links or browser back/forward.
  // Uses useLayoutEffect so the scope is updated BEFORE any child useEffect fires
  // (e.g., list pages that fetch data based on scopeWorkspaceId).
  // /iam/workspaces/:id is a platform-level detail page — scope stays null.
  // /iam/workspaces/:id/<sub-resource> activates workspace scope.
  // /iam/workspaces/:id/namespaces/:nsId/<sub-resource> activates namespace scope.
  useLayoutEffect(() => {
    const segs = location.pathname.split("/").filter(Boolean)
    // Skip module prefix (e.g. "iam", "dashboard")
    const s = isModulePrefix(segs[0]) ? segs.slice(1) : segs
    let urlWsId: string | null = null
    let urlNsId: string | null = null

    if (s[0] === "workspaces" && s[1] && s.length > 2) {
      urlWsId = s[1]
      if (s[2] === "namespaces" && s[3] && s.length > 4) {
        urlNsId = s[3]
      }
    }

    if (urlWsId !== scopeWorkspaceId || urlNsId !== scopeNamespaceId) {
      setScope(urlWsId, urlNsId)
    }
  }, [location.pathname]) // eslint-disable-line react-hooks/exhaustive-deps
  const navGroups = useMemo(
    () => buildNavGroups(scopeWorkspaceId, scopeNamespaceId),
    [scopeWorkspaceId, scopeNamespaceId],
  )
  const permissions = usePermissionStore((s) => s.permissions)
  const homePath = useMemo(
    () => permissions ? getDefaultPath(permissions) : "/dashboard/overview",
    [permissions],
  )

  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (!isAuthenticated()) {
      startAuthFlow()
      return
    }
    ;(async () => {
      await fetchUser()
      const u = useAuthStore.getState().user
      if (u?.sub) {
        await fetchPermissions(u.sub)
      }
      setReady(true)
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (!isAuthenticated() || !ready) {
    return null
  }

  // Redirect to 403 if user has zero permissions (or fetchPermissions was never called)
  if (!permissions) {
    return <Navigate to="/error?status=403" replace />
  }
  const hasAny =
    permissions.isPlatformAdmin ||
    (permissions.platform?.length ?? 0) > 0 ||
    Object.keys(permissions.workspaces ?? {}).length > 0 ||
    Object.keys(permissions.namespaces ?? {}).length > 0
  if (!hasAny) {
    return <Navigate to="/error?status=403" replace />
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen">
        <aside className="bg-sidebar text-sidebar-foreground flex w-60 flex-col border-r">
          <div className="flex h-14 items-center border-b px-4">
            <Link to={homePath} className="flex items-center gap-2 font-semibold">
              <LayoutDashboard className="h-5 w-5" />
              <span>LCP Console</span>
            </Link>
          </div>
          <div className="border-b px-1 py-1.5">
            <ScopeSelector />
          </div>
          <nav className="flex-1 space-y-3 p-2">
            {navGroups.map((group, gi) => {
              const visibleItems = group.items.filter(
                (item) => !item.permission || hasPermission(item.permission, item.permissionScope),
              )
              if (visibleItems.length === 0) return null
              return (
                <div key={group.labelKey ?? `group-${gi}`}>
                  {group.labelKey && (
                    <div className="text-muted-foreground px-3 pb-1 pt-2 text-sm font-semibold">
                      {t(group.labelKey)}
                    </div>
                  )}
                  <div className="space-y-0.5">
                    {visibleItems.map((item) => (
                      <Link
                        key={item.to}
                        to={item.to}
                        className={cn(
                          "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                          location.pathname.startsWith(item.to)
                            ? "bg-sidebar-accent text-sidebar-accent-foreground"
                            : "text-muted-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground",
                        )}
                      >
                        <item.icon className="h-4 w-4" />
                        {t(item.labelKey)}
                      </Link>
                    ))}
                  </div>
                </div>
              )
            })}
          </nav>
        </aside>
        <div className="flex flex-1 flex-col">
          <header className="flex h-14 items-center justify-between border-b px-6">
            <AppBreadcrumb />
            <div className="ml-auto flex items-center gap-2">
              <a
                href="/api-docs"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:bg-accent hover:text-accent-foreground inline-flex h-9 w-9 items-center justify-center rounded-md text-sm font-medium transition-colors"
                title={t("nav.apiDocs")}
              >
                <FileText className="h-4 w-4" />
              </a>
              <LanguageSwitcher />
              <UserMenu />
            </div>
          </header>
          <main className="flex-1 overflow-auto">
            <Outlet />
          </main>
        </div>
      </div>
    </TooltipProvider>
  )
}
