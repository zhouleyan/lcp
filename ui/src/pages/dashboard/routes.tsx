import type { RouteObject } from "react-router"
import { PlatformOverviewPage } from "./overview"

export const dashboardRoutes: RouteObject[] = [
  { path: "overview", element: <PlatformOverviewPage /> },
]
