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
  "common.edit": "Edit",
  "common.cancel": "Cancel",
  "common.confirm": "Confirm",
  "common.save": "Save",
  "common.search": "Search",
  "common.actions": "Actions",
  "common.active": "Active",
  "common.inactive": "Inactive",
  "common.all": "All",
  "common.phone": "Phone",
  "common.password": "Password",
  "common.previous": "Previous",
  "common.next": "Next",
  "common.page": "Page {page} of {total}",

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
  "nav.iam": "IAM",
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
  "user.edit": "Edit User",
  "user.noData": "No users found.",
  "user.username": "Username",
  "user.email": "Email",
  "user.searchPlaceholder": "Search username, email, phone, display name...",
  "common.updated": "Updated",
  "user.deleteConfirm": "Are you sure you want to delete user \"{name}\"? This action cannot be undone.",
  "user.batchDelete": "Batch Delete",
  "user.batchDeleteConfirm": "Are you sure you want to delete {count} selected users? This action cannot be undone.",

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

  // login errors
  "login.error.invalidCredentials": "Invalid username or password",
  "login.error.accountInactive": "Account has been deactivated",
  "login.error.failed": "Login failed, please try again",

  // api errors
  "api.error.badRequest": "Bad request",
  "api.error.notFound": "{resource} not found",
  "api.error.conflict": "{resource} already exists",
  "api.error.oldPasswordIncorrect": "Current password is incorrect",
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

export default enUS
