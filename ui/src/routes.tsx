import { Navigate, type RouteObject } from "react-router"
import RootLayout from "@/layouts/root-layout"
import LoginPage from "@/pages/login"
import ApiDocsPage from "@/pages/api-docs"
import AuthCallbackPage from "@/pages/auth-callback"
import ErrorPage from "@/pages/error"
import { dashboardRoutes } from "@/pages/dashboard/routes"
import { iamRoutes } from "@/pages/iam/routes"
import { auditRoutes } from "@/pages/audit/routes"

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
      { index: true, element: <Navigate to="/dashboard/overview" replace /> },
      {
        path: "dashboard",
        children: [
          { index: true, element: <Navigate to="/dashboard/overview" replace /> },
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
    ],
  },
  {
    path: "*",
    element: <Navigate to="/dashboard/overview" replace />,
  },
]
