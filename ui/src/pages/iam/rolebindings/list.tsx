import { useMemo } from "react"
import {
  listRoleBindings, createRoleBinding, deleteRoleBinding, deleteRoleBindings, listRoles,
} from "@/api/iam/rbac"
import { RoleBindingListView, type RoleBindingListConfig } from "@/components/rolebinding-list-view"

export default function RoleBindingListPage() {
  const config = useMemo<RoleBindingListConfig>(() => ({
    listBindings: (params) => listRoleBindings(params),
    createBinding: (data) => createRoleBinding(data),
    deleteBinding: (id) => deleteRoleBinding(id),
    deleteBindings: (ids) => deleteRoleBindings(ids),
    listRoles: (params) => listRoles(params),
    permCreate: "iam:rolebindings:create",
    permDelete: "iam:rolebindings:delete",
    scope: "platform",
  }), [])

  return <RoleBindingListView config={config} />
}
