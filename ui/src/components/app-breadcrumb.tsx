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
}

export function AppBreadcrumb() {
  const { t } = useTranslation()
  const location = useLocation()
  const workspaceName = useWorkspaceStore((s) => s.currentWorkspaceName)

  const segments = location.pathname.split("/").filter(Boolean)

  // Don't render breadcrumb on root / index
  if (segments.length === 0) return null

  const items: BreadcrumbEntry[] = []
  let pathAccum = ""

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
        // For namespace IDs, just show the ID (no store for namespace name yet)
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
