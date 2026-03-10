import type { RouteObject } from "react-router"
import WorkspaceLayout from "@/layouts/workspace-layout"
import { PlatformOverviewPage, WorkspaceOverviewPage, NamespaceOverviewPage } from "./overview"

export const dashboardRoutes: RouteObject[] = [
  { path: "overview", element: <PlatformOverviewPage /> },
  {
    path: "workspaces/:workspaceId",
    element: <WorkspaceLayout />,
    children: [
      { path: "overview", element: <WorkspaceOverviewPage /> },
      { path: "namespaces/:namespaceId/overview", element: <NamespaceOverviewPage /> },
    ],
  },
]
