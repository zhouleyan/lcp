import { useMemo } from "react"
import { useParams, Navigate } from "react-router"
import {
  listWorkspaceRoleBindings, createWorkspaceRoleBinding,
  deleteWorkspaceRoleBinding, deleteWorkspaceRoleBindings, listWorkspaceRoles,
} from "@/api/iam/rbac"
import { usePermission } from "@/hooks/use-permission"
import { usePermissionStore } from "@/stores/permission-store"
import { RoleBindingListView, type RoleBindingListConfig } from "@/components/rolebinding-list-view"

export default function WorkspaceRoleBindingsTab() {
  const workspaceId = useParams().workspaceId!
  const { hasPermission } = usePermission()
  const permissionsLoaded = usePermissionStore((s) => s.permissions) !== null

  const config = useMemo<RoleBindingListConfig>(() => ({
    listBindings: (params) => listWorkspaceRoleBindings(workspaceId, params),
    createBinding: (data) => createWorkspaceRoleBinding(workspaceId, data),
    deleteBinding: (id) => deleteWorkspaceRoleBinding(workspaceId, id),
    deleteBindings: (ids) => deleteWorkspaceRoleBindings(workspaceId, ids),
    listRoles: (params) => listWorkspaceRoles(workspaceId, params),
    permCreate: "iam:workspaces:rolebindings:create",
    permDelete: "iam:workspaces:rolebindings:delete",
    scope: "workspace",
    scopeParams: { workspaceId },
  }), [workspaceId])

  if (permissionsLoaded && !hasPermission("iam:workspaces:rolebindings:list", { workspaceId })) {
    return <Navigate to="/" replace />
  }

  return <RoleBindingListView config={config} />
}
