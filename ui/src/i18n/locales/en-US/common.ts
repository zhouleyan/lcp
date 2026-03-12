import type { Messages } from "../../types"

const common: Messages = {
  // common
  "common.name": "Name",
  "common.displayName": "Display Name",
  "common.status": "Status",
  "common.created": "Created",
  "common.total": "{count} total",
  "common.delete": "Delete",
  "common.description": "Description",
  "common.edit": "Edit",
  "common.cancel": "Cancel",
  "common.confirm": "Confirm",
  "common.save": "Save",
  "common.search": "Search",
  "common.actions": "Actions",
  "common.active": "Active",
  "common.inactive": "Inactive",
  "common.all": "All",
  "common.noPermission": "You don't have permission to access this page",
  "common.phone": "Phone",
  "common.password": "Password",
  "common.previous": "Previous",
  "common.next": "Next",
  "common.pageSize": "Per page",
  "common.page": "Page {page} of {total}",
  "common.noSearchResults": "No matching results found.",
  "common.reset": "Reset",
  "common.updated": "Updated",

  // auth
  "auth.authenticating": "Authenticating...",
  "auth.missingCode": "Missing authorization code",

  // login
  "login.title": "LCP Console",
  "login.username": "Username",
  "login.password": "Password",
  "login.usernamePlaceholder": "Enter username",
  "login.passwordPlaceholder": "Enter password",
  "login.signIn": "Sign In",

  // nav
  "nav.overview": "Overview",
  "nav.dashboard": "Dashboard",
  "nav.iam": "IAM",
  "nav.workspaces": "Workspaces",
  "nav.namespaces": "Namespaces",
  "nav.users": "Users",
  "nav.roles": "Roles",
  "nav.audit": "Audit",
  "nav.auditLogs": "Audit Logs",
  "nav.rolebindings": "Role Bindings",
  "nav.infra": "Infrastructure",
  "nav.hosts": "Hosts",
  "nav.environments": "Environments",
  "nav.regions": "Regions",
  "nav.sites": "Sites",
  "nav.locations": "Locations",
  "nav.apiDocs": "API Docs",

  // overview
  "overview.platform.title": "Platform Overview",
  "overview.platform.desc": "View overall platform resource summary",
  "overview.workspace.title": "Workspace Overview",
  "overview.workspace.desc": "View resource summary for the current workspace",
  "overview.namespace.title": "Namespace Overview",
  "overview.namespace.desc": "View resource summary for the current namespace",
  "overview.forbidden": "You don't have permission to view overview statistics. Please use the sidebar to navigate to pages you have access to.",

  // scope selector
  "scope.allWorkspaces": "All Workspaces",
  "scope.allNamespaces": "All Namespaces",
  "scope.selectWorkspace": "Select workspace",
  "scope.selectNamespace": "Select namespace",

  // permission verb wildcards
  "perm.group.all": "All Permissions",
  "perm.verb.list": "All list (*:list)",
  "perm.verb.get": "All get (*:get)",
  "perm.verb.create": "All create (*:create)",
  "perm.verb.update": "All update (*:update)",
  "perm.verb.patch": "All patch (*:patch)",
  "perm.verb.delete": "All delete (*:delete)",
  "perm.verb.deleteCollection": "All batch delete (*:deleteCollection)",

  // permission verb groups
  "perm.verbGroup.read": "Read",
  "perm.verbGroup.create": "Create",
  "perm.verbGroup.update": "Update",
  "perm.verbGroup.delete": "Delete",

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
  "error.switchAccount": "Switch Account",

  // login errors
  "login.error.invalidCredentials": "Invalid username or password",
  "login.error.accountInactive": "Account has been deactivated",
  "login.error.sessionExpired": "Session expired, redirecting...",
  "login.error.failed": "Login failed, please try again",

  // api errors
  "api.error.badRequest": "Bad request",
  "api.error.notFound": "{resource} not found",
  "api.error.conflict": "{resource} already exists",
  "api.error.memberLimitExceeded": "Member limit exceeded for this namespace",
  "api.error.cannotDeleteWorkspace": "Cannot delete workspace: it still contains namespaces, please delete all namespaces first",
  "api.error.cannotDeleteNamespace": "Cannot delete namespace: it still contains members, please remove all members first",
  "api.error.cannotRemoveOwner": "Cannot remove the owner from this resource",
  "api.error.oldPasswordIncorrect": "Current password is incorrect",
  "api.error.forbidden": "You do not have permission to perform this action",
  "api.error.internalError": "Internal server error, please try again later",

  // validation errors
  "api.validation.required": "{field} is required",
  "api.validation.username.format": "Username must be 3-50 characters of letters, digits, or underscores",
  "api.validation.email.format": "Please enter a valid email address",
  "api.validation.phone.format": "Please enter a valid mobile number (e.g. 13800138000)",
  "api.validation.password.length": "Password must be 8-128 characters",
  "api.validation.password.uppercase": "Password must contain at least one uppercase letter",
  "api.validation.password.lowercase": "Password must contain at least one lowercase letter",
  "api.validation.password.digit": "Password must contain at least one digit",
  "api.validation.name.format": "Name must be 3-50 lowercase letters, digits, or hyphens",
  "api.validation.rackCapacity.min": "Rack capacity must be >= 0",
  "api.validation.status.format": "Status must be 'active' or 'inactive'",
  "api.validation.username.taken": "This username is already taken",
  "api.validation.email.taken": "This email is already taken",
  "api.validation.phone.taken": "This phone number is already taken",
  "api.validation.password.hint": "8-128 characters, must include uppercase, lowercase, and a digit",

  // action feedback
  "action.createSuccess": "Created successfully",
  "action.updateSuccess": "Updated successfully",
  "action.deleteSuccess": "Deleted successfully",
  "action.changePasswordSuccess": "Password changed successfully",

  // user menu
  "userMenu.profile": "Profile",
  "userMenu.changePassword": "Change Password",
  "userMenu.logout": "Sign Out",
  "userMenu.oldPassword": "Current Password",
  "userMenu.newPassword": "New Password",
  "userMenu.confirmPassword": "Confirm New Password",
  "userMenu.passwordMismatch": "Passwords do not match",
  "userMenu.passwordSameAsOld": "New password must be different from current password",
}

export default common
