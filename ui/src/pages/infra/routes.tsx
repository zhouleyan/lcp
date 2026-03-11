import { Navigate, type RouteObject } from "react-router"
import HostListPage from "./hosts/list"
import HostDetailPage from "./hosts/detail"
import EnvironmentListPage from "./environments/list"
import EnvironmentDetailPage from "./environments/detail"

export const infraRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/infra/hosts" replace /> },
  { path: "hosts", element: <HostListPage /> },
  { path: "hosts/:hostId", element: <HostDetailPage /> },
  { path: "environments", element: <EnvironmentListPage /> },
  { path: "environments/:environmentId", element: <EnvironmentDetailPage /> },
]
