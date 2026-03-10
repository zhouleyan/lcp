import { useEffect, useMemo } from "react"
import { Link, Outlet, useLocation } from "react-router"
import {
  LayoutDashboard,
  Users,
  Building2,
  FolderKanban,
  FileText,
  Shield,
  Home,
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
import { usePermission } from "@/hooks/use-permission"
import { useScopeStore } from "@/stores/scope-store"

interface NavItem {
  to: string
  labelKey: string
  icon: React.ComponentType<{ className?: string }>
  permission?: string
}

interface NavGroup {
  labelKey?: string
  items: NavItem[]
}

function getOverviewPath(scopeWorkspaceId: string | null, scopeNamespaceId: string | null): string {
  if (scopeWorkspaceId && scopeNamespaceId) {
    return `/iam/workspaces/${scopeWorkspaceId}/namespaces/${scopeNamespaceId}/overview`
  }
  if (scopeWorkspaceId) {
    return `/iam/workspaces/${scopeWorkspaceId}/overview`
  }
  return "/iam/overview"
}

function buildNavGroups(scopeWorkspaceId: string | null, scopeNamespaceId: string | null): NavGroup[] {
  if (scopeWorkspaceId && scopeNamespaceId) {
    const prefix = `/iam/workspaces/${scopeWorkspaceId}/namespaces/${scopeNamespaceId}`
    return [
      { items: [{ to: `${prefix}/overview`, labelKey: "nav.overview", icon: Home }] },
      {
        labelKey: "nav.iam",
        items: [
          { to: `${prefix}/users`, labelKey: "nav.users", icon: Users },
          { to: `${prefix}/roles`, labelKey: "nav.roles", icon: Shield },
        ],
      },
    ]
  }
  if (scopeWorkspaceId) {
    const prefix = `/iam/workspaces/${scopeWorkspaceId}`
    return [
      { items: [{ to: `${prefix}/overview`, labelKey: "nav.overview", icon: Home }] },
      {
        labelKey: "nav.iam",
        items: [
          { to: `${prefix}/namespaces`, labelKey: "nav.namespaces", icon: FolderKanban },
          { to: `${prefix}/users`, labelKey: "nav.users", icon: Users },
          { to: `${prefix}/roles`, labelKey: "nav.roles", icon: Shield },
        ],
      },
    ]
  }
  return [
    { items: [{ to: "/iam/overview", labelKey: "nav.overview", icon: Home }] },
    {
      labelKey: "nav.iam",
      items: [
        { to: "/iam/workspaces", labelKey: "nav.workspaces", icon: Building2 },
        { to: "/iam/namespaces", labelKey: "nav.namespaces", icon: FolderKanban },
        { to: "/iam/users", labelKey: "nav.users", icon: Users, permission: "iam:users:list" },
        { to: "/iam/roles", labelKey: "nav.roles", icon: Shield, permission: "iam:roles:list" },
      ],
    },
  ]
}

export default function RootLayout() {
  const location = useLocation()
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const fetchUser = useAuthStore((s) => s.fetchUser)
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions)
  const permissionsLoaded = usePermissionStore((s) => s.permissions !== null)
  const { hasPermission } = usePermission()
  const scopeWorkspaceId = useScopeStore((s) => s.workspaceId)
  const scopeNamespaceId = useScopeStore((s) => s.namespaceId)
  const setScope = useScopeStore((s) => s.setScope)

  // Sync scope store from URL when navigating via links or browser back/forward.
  // /iam/workspaces/:id is a platform-level detail page — scope stays null.
  // /iam/workspaces/:id/<sub-resource> activates workspace scope.
  // /iam/workspaces/:id/namespaces/:nsId/<sub-resource> activates namespace scope.
  useEffect(() => {
    const segs = location.pathname.split("/").filter(Boolean)
    // Skip module prefix (e.g. "iam")
    const s = segs[0] === "iam" ? segs.slice(1) : segs
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
  const overviewPath = useMemo(
    () => getOverviewPath(scopeWorkspaceId, scopeNamespaceId),
    [scopeWorkspaceId, scopeNamespaceId],
  )

  useEffect(() => {
    if (!isAuthenticated()) {
      startAuthFlow()
    } else {
      fetchUser()
    }
  }, [])

  useEffect(() => {
    if (user?.sub) {
      fetchPermissions(user.sub)
    }
  }, [user?.sub, fetchPermissions])

  if (!isAuthenticated()) {
    return null
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen">
        <aside className="bg-sidebar text-sidebar-foreground flex w-60 flex-col border-r">
          <div className="flex h-14 items-center border-b px-4">
            <Link to={overviewPath} className="flex items-center gap-2 font-semibold">
              <LayoutDashboard className="h-5 w-5" />
              <span>LCP Console</span>
            </Link>
          </div>
          <div className="border-b px-1 py-1.5">
            <ScopeSelector />
          </div>
          <nav className="flex-1 space-y-3 p-2">
            {navGroups.map((group, gi) => (
              <div key={group.labelKey ?? `group-${gi}`}>
                {group.labelKey && (
                  <div className="text-muted-foreground px-3 pb-1 pt-2 text-sm font-semibold">
                    {t(group.labelKey)}
                  </div>
                )}
                <div className="space-y-0.5">
                  {group.items
                    .filter(
                      (item) =>
                        !item.permission || !permissionsLoaded || hasPermission(item.permission),
                    )
                    .map((item) => (
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
            ))}
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
