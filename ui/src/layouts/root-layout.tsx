import { useEffect, useLayoutEffect, useMemo, useState } from "react"
import { Link, Navigate, Outlet, useLocation } from "react-router"
import { LayoutDashboard, FileText } from "lucide-react"
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
import { NAV_ITEMS, buildScopedPath, buildPermScope } from "@/lib/nav-config"
import type { ScopeLevel } from "@/lib/nav-config"

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

function buildNavGroups(wsId: string | null, nsId: string | null): NavGroup[] {
  const scopeLevel: ScopeLevel = (wsId && nsId) ? "namespace" : wsId ? "workspace" : "platform"
  const scope = buildPermScope(wsId ?? undefined, nsId ?? undefined)

  const groups: NavGroup[] = []
  let currentGroup: NavGroup | null = null

  for (const item of NAV_ITEMS) {
    if (!item.scopes.includes(scopeLevel)) continue

    const navItem: NavItem = {
      to: buildScopedPath(item.resource, wsId, nsId),
      labelKey: item.labelKey,
      icon: item.icon,
      permission: item.permission,
      permissionScope: scope,
    }

    if (!item.group) {
      // Standalone item (e.g. overview) — its own group
      groups.push({ items: [navItem] })
      currentGroup = null
    } else if (currentGroup != null && currentGroup.labelKey === item.group) {
      currentGroup.items.push(navItem)
    } else {
      currentGroup = { labelKey: item.group, items: [navItem] }
      groups.push(currentGroup)
    }
  }

  return groups
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
