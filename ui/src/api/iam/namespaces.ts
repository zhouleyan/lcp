import { iamApi } from "./client"
import { apiRequest } from "../client"
import type { Namespace, NamespaceList, ListParams } from "../types"

export async function listNamespaces(params?: ListParams): Promise<NamespaceList> {
  return apiRequest(iamApi.get("namespaces", { searchParams: params as Record<string, string> }).json())
}

export async function listWorkspaceNamespaces(workspaceId: string, params?: ListParams): Promise<NamespaceList> {
  return apiRequest(iamApi.get(`workspaces/${workspaceId}/namespaces`, { searchParams: params as Record<string, string> }).json())
}

export async function getNamespace(id: string): Promise<Namespace> {
  return apiRequest(iamApi.get(`namespaces/${id}`).json())
}

export async function createNamespace(data: Pick<Namespace, "metadata" | "spec">): Promise<Namespace> {
  return apiRequest(iamApi.post("namespaces", { json: data }).json())
}

export async function createWorkspaceNamespace(workspaceId: string, data: Pick<Namespace, "metadata" | "spec">): Promise<Namespace> {
  return apiRequest(iamApi.post(`workspaces/${workspaceId}/namespaces`, { json: data }).json())
}

export async function updateNamespace(id: string, data: Pick<Namespace, "metadata" | "spec">): Promise<Namespace> {
  return apiRequest(iamApi.put(`namespaces/${id}`, { json: data }).json())
}

export async function deleteNamespace(id: string): Promise<void> {
  await apiRequest(iamApi.delete(`namespaces/${id}`).json())
}

export async function deleteNamespaces(ids: string[]): Promise<void> {
  await apiRequest(iamApi.delete("namespaces", { json: { ids } }).json())
}
