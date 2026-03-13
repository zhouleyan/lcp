import { Navigate, type RouteObject } from "react-router"
import EndpointListPage from "./endpoints/list"

export const o11yRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/o11y/endpoints" replace /> },
  { path: "endpoints", element: <EndpointListPage /> },
]
