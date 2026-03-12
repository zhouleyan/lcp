import { Navigate, type RouteObject } from "react-router"
import RootLayout from "@/layouts/root-layout"
import LoginPage from "@/pages/login"
import ApiDocsPage from "@/pages/api-docs"
import AuthCallbackPage from "@/pages/auth-callback"
import ErrorPage from "@/pages/error"
import { dashboardRoutes } from "@/pages/dashboard/routes"
import { iamRoutes } from "@/pages/iam/routes"
import { auditRoutes } from "@/pages/audit/routes"
import { infraRoutes } from "@/pages/infra/routes"
import { networkRoutes } from "@/pages/network/routes"
import { usePermissionStore } from "@/stores/permission-store"
import { getDefaultPath } from "@/hooks/use-permission"

function DefaultRedirect() {
  const permissions = usePermissionStore((s) => s.permissions)
  const target = permissions ? getDefaultPath(permissions) : "/dashboard/overview"
  return <Navigate to={target} replace />
}

export const routes: RouteObject[] = [
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/api-docs",
    element: <ApiDocsPage />,
  },
  {
    path: "/auth/callback",
    element: <AuthCallbackPage />,
  },
  {
    path: "/error",
    element: <ErrorPage />,
  },
  {
    path: "/",
    element: <RootLayout />,
    children: [
      { index: true, element: <DefaultRedirect /> },
      {
        path: "dashboard",
        children: [
          { index: true, element: <DefaultRedirect /> },
          ...dashboardRoutes,
        ],
      },
      {
        path: "iam",
        children: iamRoutes,
      },
      {
        path: "audit",
        children: [
          { index: true, element: <Navigate to="/audit/logs" replace /> },
          ...auditRoutes,
        ],
      },
      {
        path: "infra",
        children: infraRoutes,
      },
      {
        path: "network",
        children: networkRoutes,
      },
    ],
  },
  {
    path: "*",
    element: <DefaultRedirect />,
  },
]
