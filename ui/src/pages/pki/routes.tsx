import { Navigate, type RouteObject } from "react-router"
import CertificateListPage from "./certificates/list"
import CertificateDetailPage from "./certificates/detail"

export const pkiRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="/pki/certificates" replace /> },
  { path: "certificates", element: <CertificateListPage /> },
  { path: "certificates/:certificateId", element: <CertificateDetailPage /> },
]
