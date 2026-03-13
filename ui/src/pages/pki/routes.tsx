import { Navigate, type RouteObject } from "react-router"
import CertificateListPage from "./certificates/list"

export const pkiRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/pki/certificates" replace /> },
  { path: "certificates", element: <CertificateListPage /> },
]
