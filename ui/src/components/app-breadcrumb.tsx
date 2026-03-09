import React from "react"
import { Link, useLocation } from "react-router"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { useTranslation } from "@/i18n"
import { useWorkspaceStore } from "@/stores/workspace-store"

interface BreadcrumbEntry {
  label: string
  href?: string
}

const routeLabelKeys: Record<string, string> = {
  workspaces: "nav.workspaces",
  namespaces: "nav.namespaces",
  users: "nav.users",
  roles: "nav.roles",
}

export function AppBreadcrumb() {
  const { t } = useTranslation()
  const location = useLocation()
  const workspaceName = useWorkspaceStore((s) => s.currentWorkspaceName)

  const rawSegments = location.pathname.split("/").filter(Boolean)

  // Don't render breadcrumb on root / index
  if (rawSegments.length === 0) return null

  // For scoped routes, strip the scope prefix from breadcrumb display
  // but preserve it in link hrefs so navigation stays within scope.
  // e.g. /workspaces/4/namespaces/4/roles/35
  //   → display: Roles > 35
  //   → hrefs:   /workspaces/4/namespaces/4/roles, (current page)
  let segments = rawSegments
  let scopePrefix = ""
  if (rawSegments[0] === "workspaces" && rawSegments[1]) {
    // /workspaces/:id/namespaces/:nsId/... → strip first 4, show from sub-resource
    if (rawSegments[2] === "namespaces" && rawSegments[3] && rawSegments.length > 4) {
      scopePrefix = `/${rawSegments.slice(0, 4).join("/")}`
      segments = rawSegments.slice(4)
    }
    // /workspaces/:id/... → strip first 2, show from resource onward
    else if (rawSegments.length > 2) {
      scopePrefix = `/${rawSegments.slice(0, 2).join("/")}`
      segments = rawSegments.slice(2)
    }
  }

  const items: BreadcrumbEntry[] = []
  let pathAccum = scopePrefix

  for (let i = 0; i < segments.length; i++) {
    const seg = segments[i]
    pathAccum += "/" + seg
    const isLast = i === segments.length - 1

    if (routeLabelKeys[seg]) {
      items.push({
        label: t(routeLabelKeys[seg]),
        href: isLast ? undefined : pathAccum,
      })
    } else {
      // Dynamic segment (e.g. workspace ID, namespace ID)
      const parentSeg = segments[i - 1]
      if (parentSeg === "workspaces") {
        items.push({
          label: workspaceName ?? seg,
          href: isLast ? undefined : pathAccum,
        })
      } else if (parentSeg === "namespaces") {
        items.push({
          label: seg,
          href: isLast ? undefined : pathAccum,
        })
      } else {
        items.push({ label: seg })
      }
    }
  }

  if (items.length === 0) return null

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {items.map((item, i) => {
          const isLast = i === items.length - 1
          return (
            <React.Fragment key={i}>
              {i > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{item.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link to={item.href!}>{item.label}</Link>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </React.Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
