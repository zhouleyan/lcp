import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Environment, EnvironmentList, HostList, ListParams } from "../types"

// --- Platform-level ---

export async function listEnvironments(params?: ListParams): Promise<EnvironmentList> {
  return apiRequest(infraApi.get("environments", { searchParams: params as Record<string, string> }).json())
}

export async function getEnvironment(id: string): Promise<Environment> {
  return apiRequest(infraApi.get(`environments/${id}`).json())
}

export async function createEnvironment(data: Pick<Environment, "metadata" | "spec">): Promise<Environment> {
  return apiRequest(infraApi.post("environments", { json: data }).json())
}

export async function updateEnvironment(
  id: string,
  data: Pick<Environment, "metadata" | "spec">,
): Promise<Environment> {
  return apiRequest(infraApi.put(`environments/${id}`, { json: data }).json())
}

export async function patchEnvironment(
  id: string,
  data: Partial<Pick<Environment, "metadata" | "spec">>,
): Promise<Environment> {
  return apiRequest(infraApi.patch(`environments/${id}`, { json: data }).json())
}

export async function deleteEnvironment(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`environments/${id}`).json())
}

export async function deleteEnvironments(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("environments", { json: { ids } }).json())
}

export async function getEnvironmentHosts(id: string, params?: ListParams): Promise<HostList> {
  return apiRequest(
    infraApi.get(`environments/${id}:hosts`, { searchParams: params as Record<string, string> }).json(),
  )
}

// --- Workspace-level ---

export async function listWorkspaceEnvironments(wsId: string, params?: ListParams): Promise<EnvironmentList> {
  return apiRequest(
    infraApi.get(`workspaces/${wsId}/environments`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function getWorkspaceEnvironment(wsId: string, envId: string): Promise<Environment> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/environments/${envId}`).json())
}

export async function createWorkspaceEnvironment(
  wsId: string,
  data: Pick<Environment, "metadata" | "spec">,
): Promise<Environment> {
  return apiRequest(infraApi.post(`workspaces/${wsId}/environments`, { json: data }).json())
}

export async function updateWorkspaceEnvironment(
  wsId: string,
  envId: string,
  data: Pick<Environment, "metadata" | "spec">,
): Promise<Environment> {
  return apiRequest(infraApi.put(`workspaces/${wsId}/environments/${envId}`, { json: data }).json())
}

export async function deleteWorkspaceEnvironment(wsId: string, envId: string): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/environments/${envId}`).json())
}

export async function deleteWorkspaceEnvironments(wsId: string, ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/environments`, { json: { ids } }).json())
}

export async function getWorkspaceEnvironmentHosts(
  wsId: string,
  envId: string,
  params?: ListParams,
): Promise<HostList> {
  return apiRequest(
    infraApi
      .get(`workspaces/${wsId}/environments/${envId}:hosts`, { searchParams: params as Record<string, string> })
      .json(),
  )
}

// --- Namespace-level ---

export async function listNamespaceEnvironments(
  wsId: string,
  nsId: string,
  params?: ListParams,
): Promise<EnvironmentList> {
  return apiRequest(
    infraApi
      .get(`workspaces/${wsId}/namespaces/${nsId}/environments`, {
        searchParams: params as Record<string, string>,
      })
      .json(),
  )
}

export async function getNamespaceEnvironment(wsId: string, nsId: string, envId: string): Promise<Environment> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/namespaces/${nsId}/environments/${envId}`).json())
}

export async function createNamespaceEnvironment(
  wsId: string,
  nsId: string,
  data: Pick<Environment, "metadata" | "spec">,
): Promise<Environment> {
  return apiRequest(infraApi.post(`workspaces/${wsId}/namespaces/${nsId}/environments`, { json: data }).json())
}

export async function updateNamespaceEnvironment(
  wsId: string,
  nsId: string,
  envId: string,
  data: Pick<Environment, "metadata" | "spec">,
): Promise<Environment> {
  return apiRequest(
    infraApi.put(`workspaces/${wsId}/namespaces/${nsId}/environments/${envId}`, { json: data }).json(),
  )
}

export async function deleteNamespaceEnvironment(wsId: string, nsId: string, envId: string): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/namespaces/${nsId}/environments/${envId}`).json())
}

export async function deleteNamespaceEnvironments(wsId: string, nsId: string, ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/namespaces/${nsId}/environments`, { json: { ids } }).json())
}

export async function getNamespaceEnvironmentHosts(
  wsId: string,
  nsId: string,
  envId: string,
  params?: ListParams,
): Promise<HostList> {
  return apiRequest(
    infraApi
      .get(`workspaces/${wsId}/namespaces/${nsId}/environments/${envId}:hosts`, {
        searchParams: params as Record<string, string>,
      })
      .json(),
  )
}
