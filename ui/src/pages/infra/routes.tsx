import { Navigate, type RouteObject } from "react-router"
import HostListPage from "./hosts/list"
import HostDetailPage from "./hosts/detail"
import EnvironmentListPage from "./environments/list"
import EnvironmentDetailPage from "./environments/detail"
import RegionListPage from "./regions/list"
import RegionDetailPage from "./regions/detail"
import SiteListPage from "./sites/list"
import SiteDetailPage from "./sites/detail"
import LocationListPage from "./locations/list"
import LocationDetailPage from "./locations/detail"
import RackListPage from "./racks/list"
import RackDetailPage from "./racks/detail"

export const infraRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/infra/hosts" replace /> },
  // Platform-level
  { path: "hosts", element: <HostListPage /> },
  { path: "hosts/:hostId", element: <HostDetailPage /> },
  { path: "environments", element: <EnvironmentListPage /> },
  { path: "environments/:environmentId", element: <EnvironmentDetailPage /> },
  // Workspace-level
  { path: "workspaces/:workspaceId/hosts", element: <HostListPage /> },
  { path: "workspaces/:workspaceId/hosts/:hostId", element: <HostDetailPage /> },
  { path: "workspaces/:workspaceId/environments", element: <EnvironmentListPage /> },
  { path: "workspaces/:workspaceId/environments/:environmentId", element: <EnvironmentDetailPage /> },
  // Namespace-level
  { path: "workspaces/:workspaceId/namespaces/:namespaceId/hosts", element: <HostListPage /> },
  { path: "workspaces/:workspaceId/namespaces/:namespaceId/hosts/:hostId", element: <HostDetailPage /> },
  { path: "workspaces/:workspaceId/namespaces/:namespaceId/environments", element: <EnvironmentListPage /> },
  { path: "workspaces/:workspaceId/namespaces/:namespaceId/environments/:environmentId", element: <EnvironmentDetailPage /> },
  // CMDB - Platform-only
  { path: "regions", element: <RegionListPage /> },
  { path: "regions/:regionId", element: <RegionDetailPage /> },
  { path: "sites", element: <SiteListPage /> },
  { path: "sites/:siteId", element: <SiteDetailPage /> },
  { path: "locations", element: <LocationListPage /> },
  { path: "locations/:locationId", element: <LocationDetailPage /> },
  { path: "racks", element: <RackListPage /> },
  { path: "racks/:rackId", element: <RackDetailPage /> },
]
