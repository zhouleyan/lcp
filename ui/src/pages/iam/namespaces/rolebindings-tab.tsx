import { useMemo } from "react"
import { useParams, Navigate } from "react-router"
import {
  listNamespaceRoleBindings, createNamespaceRoleBinding,
  deleteNamespaceRoleBinding, deleteNamespaceRoleBindings, listNamespaceRoles,
} from "@/api/iam/rbac"
import { usePermission } from "@/hooks/use-permission"
import { usePermissionStore } from "@/stores/permission-store"
import { RoleBindingListView, type RoleBindingListConfig } from "@/components/rolebinding-list-view"

export default function NamespaceRoleBindingsTab() {
  const { workspaceId, namespaceId } = useParams() as { workspaceId: string; namespaceId: string }
  const { hasPermission } = usePermission()
  const permissionsLoaded = usePermissionStore((s) => s.permissions) !== null

  const config = useMemo<RoleBindingListConfig>(() => ({
    listBindings: (params) => listNamespaceRoleBindings(workspaceId, namespaceId, params),
    createBinding: (data) => createNamespaceRoleBinding(workspaceId, namespaceId, data),
    deleteBinding: (id) => deleteNamespaceRoleBinding(workspaceId, namespaceId, id),
    deleteBindings: (ids) => deleteNamespaceRoleBindings(workspaceId, namespaceId, ids),
    listRoles: (params) => listNamespaceRoles(workspaceId, namespaceId, params),
    permCreate: "iam:namespaces:rolebindings:create",
    permDelete: "iam:namespaces:rolebindings:delete",
    scope: "namespace",
    scopeParams: { workspaceId, namespaceId },
  }), [workspaceId, namespaceId])

  if (permissionsLoaded && !hasPermission("iam:namespaces:rolebindings:list", { workspaceId, namespaceId })) {
    return <Navigate to="/" replace />
  }

  return <RoleBindingListView config={config} />
}
