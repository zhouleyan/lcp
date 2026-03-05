import { api } from "./client"
import type { Namespace, NamespaceList, ListParams } from "./types"

export async function listNamespaces(params?: ListParams): Promise<NamespaceList> {
  return api.get("namespaces", { searchParams: params as Record<string, string> }).json()
}

export async function listWorkspaceNamespaces(
  workspaceId: string,
  params?: ListParams,
): Promise<NamespaceList> {
  return api
    .get(`workspaces/${workspaceId}/namespaces`, {
      searchParams: params as Record<string, string>,
    })
    .json()
}

export async function getNamespace(id: string): Promise<Namespace> {
  return api.get(`namespaces/${id}`).json()
}

export async function createNamespace(
  data: Pick<Namespace, "metadata" | "spec">,
): Promise<Namespace> {
  return api.post("namespaces", { json: data }).json()
}

export async function updateNamespace(
  id: string,
  data: Pick<Namespace, "metadata" | "spec">,
): Promise<Namespace> {
  return api.put(`namespaces/${id}`, { json: data }).json()
}

export async function deleteNamespace(id: string): Promise<void> {
  await api.delete(`namespaces/${id}`)
}

export async function deleteNamespaces(ids: string[]): Promise<void> {
  await api.delete("namespaces", { json: { ids } })
}
