import { useEffect } from "react"
import { Link, Outlet, useLocation } from "react-router"
import {
  LayoutDashboard,
  Users,
  Building2,
  FolderKanban,
  FileText,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { TooltipProvider } from "@/components/ui/tooltip"
import { LanguageSwitcher } from "@/components/language-switcher"
import { UserMenu } from "@/components/user-menu"
import { useTranslation } from "@/i18n"
import { isAuthenticated, startAuthFlow } from "@/lib/auth"
import { useAuthStore } from "@/stores/auth-store"

interface NavGroup {
  labelKey: string
  items: { to: string; labelKey: string; icon: React.ComponentType<{ className?: string }> }[]
}

const navGroups: NavGroup[] = [
  {
    labelKey: "nav.iam",
    items: [
      { to: "/workspaces", labelKey: "nav.workspaces", icon: Building2 },
      { to: "/namespaces", labelKey: "nav.namespaces", icon: FolderKanban },
      { to: "/users", labelKey: "nav.users", icon: Users },
    ],
  },
]

export default function RootLayout() {
  const location = useLocation()
  const { t } = useTranslation()
  const fetchUser = useAuthStore((s) => s.fetchUser)

  useEffect(() => {
    if (!isAuthenticated()) {
      startAuthFlow()
    } else {
      fetchUser()
    }
  }, [])

  if (!isAuthenticated()) {
    return null
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen">
        <aside className="bg-sidebar text-sidebar-foreground flex w-60 flex-col border-r">
          <div className="flex h-14 items-center border-b px-4">
            <Link to="/" className="flex items-center gap-2 font-semibold">
              <LayoutDashboard className="h-5 w-5" />
              <span>LCP Console</span>
            </Link>
          </div>
          <nav className="flex-1 space-y-3 p-2">
            {navGroups.map((group) => (
              <div key={group.labelKey}>
                <div className="text-muted-foreground px-3 pb-1 pt-2 text-sm font-semibold">
                  {t(group.labelKey)}
                </div>
                <div className="space-y-0.5">
                  {group.items.map((item) => (
                    <Link
                      key={item.to}
                      to={item.to}
                      className={cn(
                        "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                        location.pathname.startsWith(item.to)
                          ? "bg-sidebar-accent text-sidebar-accent-foreground"
                          : "hover:bg-sidebar-accent/50",
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
          <header className="flex h-14 items-center justify-end gap-2 border-b px-6">
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
          </header>
          <main className="flex-1 overflow-auto">
            <Outlet />
          </main>
        </div>
      </div>
    </TooltipProvider>
  )
}
