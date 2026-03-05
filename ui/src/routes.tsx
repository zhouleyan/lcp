import type { RouteObject } from "react-router"
import RootLayout from "@/layouts/root-layout"
import WorkspaceLayout from "@/layouts/workspace-layout"
import LoginPage from "@/pages/login"
import ApiDocsPage from "@/pages/api-docs"
import AuthCallbackPage from "@/pages/auth-callback"
import WorkspaceListPage from "@/pages/workspaces/list"
import WorkspaceDetailPage from "@/pages/workspaces/detail"
import NamespaceListPage from "@/pages/namespaces/list"
import UserListPage from "@/pages/users/list"

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
    path: "/",
    element: <RootLayout />,
    children: [
      { index: true, element: <WorkspaceListPage /> },
      { path: "workspaces", element: <WorkspaceListPage /> },
      {
        path: "workspaces/:workspaceId",
        element: <WorkspaceLayout />,
        children: [{ index: true, element: <WorkspaceDetailPage /> }],
      },
      { path: "namespaces", element: <NamespaceListPage /> },
      { path: "users", element: <UserListPage /> },
    ],
  },
]
