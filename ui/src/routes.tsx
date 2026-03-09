import { Navigate, type RouteObject } from "react-router"
import RootLayout from "@/layouts/root-layout"
import WorkspaceLayout from "@/layouts/workspace-layout"
import LoginPage from "@/pages/login"
import ApiDocsPage from "@/pages/api-docs"
import AuthCallbackPage from "@/pages/auth-callback"
import ErrorPage from "@/pages/error"
import WorkspaceListPage from "@/pages/workspaces/list"
import WorkspaceDetailPage from "@/pages/workspaces/detail"
import WorkspaceUsersPage from "@/pages/workspaces/users"
import WorkspaceNamespacesPage from "@/pages/workspaces/namespaces-tab"
import NamespaceListPage from "@/pages/namespaces/list"
import NamespaceDetailPage from "@/pages/namespaces/detail"
import NamespaceUsersPage from "@/pages/namespaces/users"
import UserListPage from "@/pages/users/list"
import UserDetailPage from "@/pages/users/detail"
import RoleListPage from "@/pages/roles/list"
import RoleDetailPage from "@/pages/roles/detail"
import WorkspaceRolesTab from "@/pages/workspaces/roles-tab"
import NamespaceRolesTab from "@/pages/namespaces/roles-tab"

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
      { index: true, element: <WorkspaceListPage /> },
      { path: "workspaces", element: <WorkspaceListPage /> },
      {
        path: "workspaces/:workspaceId",
        element: <WorkspaceLayout />,
        children: [
          { index: true, element: <WorkspaceDetailPage /> },
          { path: "users", element: <WorkspaceUsersPage /> },
          { path: "namespaces", element: <WorkspaceNamespacesPage /> },
          { path: "roles", element: <WorkspaceRolesTab /> },
          { path: "namespaces/:namespaceId", element: <NamespaceDetailPage /> },
          { path: "namespaces/:namespaceId/users", element: <NamespaceUsersPage /> },
          { path: "namespaces/:namespaceId/roles", element: <NamespaceRolesTab /> },
        ],
      },
      { path: "namespaces", element: <NamespaceListPage /> },
      { path: "namespaces/:namespaceId", element: <NamespaceDetailPage /> },
      { path: "namespaces/:namespaceId/users", element: <NamespaceUsersPage /> },
      { path: "namespaces/:namespaceId/roles", element: <NamespaceRolesTab /> },
      { path: "users", element: <UserListPage /> },
      { path: "users/:userId", element: <UserDetailPage /> },
      { path: "roles", element: <RoleListPage /> },
      { path: "roles/:roleId", element: <RoleDetailPage /> },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]
