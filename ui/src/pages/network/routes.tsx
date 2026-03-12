import { Navigate, type RouteObject } from "react-router"
import NetworkListPage from "./networks/list"
import NetworkDetailPage from "./networks/detail"
import SubnetDetailPage from "./networks/subnet-detail"

export const networkRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/network/networks" replace /> },
  { path: "networks", element: <NetworkListPage /> },
  { path: "networks/:networkId", element: <NetworkDetailPage /> },
  { path: "networks/:networkId/subnets/:subnetId", element: <SubnetDetailPage /> },
]
