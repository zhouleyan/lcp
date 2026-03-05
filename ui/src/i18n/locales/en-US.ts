import type { Messages } from "../types"

const enUS: Messages = {
  // common
  "common.name": "Name",
  "common.displayName": "Display Name",
  "common.status": "Status",
  "common.created": "Created",
  "common.total": "{count} total",
  "common.delete": "Delete",
  "common.description": "Description",

  // login
  "login.title": "LCP Console",
  "login.username": "Username",
  "login.password": "Password",
  "login.usernamePlaceholder": "Enter username",
  "login.passwordPlaceholder": "Enter password",
  "login.signIn": "Sign In",

  // nav
  "nav.workspaces": "Workspaces",
  "nav.namespaces": "Namespaces",
  "nav.users": "Users",
  "nav.apiDocs": "API Docs",

  // workspace
  "workspace.title": "Workspaces",
  "workspace.manage": "Manage your workspaces. {count} total.",
  "workspace.create": "Create Workspace",
  "workspace.notFound": "Workspace not found.",
  "workspace.noData": "No workspaces found.",
  "workspace.backToList": "Back to Workspaces",
  "workspace.details": "Details",
  "workspace.overview": "Overview",
  "workspace.namespaces": "Namespaces",
  "workspace.members": "Members",
  "workspace.ownerId": "Owner ID",
  "workspace.namespacesComingSoon": "Namespace list coming soon.",
  "workspace.membersComingSoon": "Member management coming soon.",

  // namespace
  "namespace.title": "Namespaces",
  "namespace.manage": "Manage namespaces. {count} total.",
  "namespace.create": "Create Namespace",
  "namespace.noData": "No namespaces found.",
  "namespace.workspaceId": "Workspace ID",
  "namespace.visibility": "Visibility",

  // user
  "user.title": "Users",
  "user.manage": "Manage platform users. {count} total.",
  "user.create": "Create User",
  "user.noData": "No users found.",
  "user.username": "Username",
  "user.email": "Email",

  // error
  "error.400.title": "Bad Request",
  "error.400.desc": "The request could not be understood. Please try again.",
  "error.401.title": "Unauthorized",
  "error.401.desc": "Please sign in to continue.",
  "error.403.title": "Forbidden",
  "error.403.desc": "You don't have permission to access this page.",
  "error.404.title": "Not Found",
  "error.404.desc": "The page you are looking for does not exist.",
  "error.500.title": "Server Error",
  "error.500.desc": "Something went wrong. Please try again later.",
  "error.backHome": "Back to Home",
}

export default enUS
