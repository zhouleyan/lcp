import { infraApi } from "./client"
import { apiRequest } from "../client"
import type {
  Host,
  HostList,
  BindEnvironmentRequest,
  ListParams,
} from "../types"

// --- Platform-level ---

export async function listHosts(params?: ListParams): Promise<HostList> {
  return apiRequest(infraApi.get("hosts", { searchParams: params as Record<string, string> }).json())
}

export async function getHost(id: string): Promise<Host> {
  return apiRequest(infraApi.get(`hosts/${id}`).json())
}

export async function createHost(data: Pick<Host, "metadata" | "spec">): Promise<Host> {
  return apiRequest(infraApi.post("hosts", { json: data }).json())
}

export async function updateHost(id: string, data: Pick<Host, "metadata" | "spec">): Promise<Host> {
  return apiRequest(infraApi.put(`hosts/${id}`, { json: data }).json())
}

export async function patchHost(id: string, data: Partial<Pick<Host, "metadata" | "spec">>): Promise<Host> {
  return apiRequest(infraApi.patch(`hosts/${id}`, { json: data }).json())
}

export async function deleteHost(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`hosts/${id}`).json())
}

export async function deleteHosts(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("hosts", { json: { ids } }).json())
}

export async function bindHostEnvironment(id: string, body: BindEnvironmentRequest): Promise<void> {
  await apiRequest(infraApi.post(`hosts/${id}/bind-environment`, { json: body }).json())
}

export async function unbindHostEnvironment(id: string): Promise<void> {
  await apiRequest(infraApi.post(`hosts/${id}/unbind-environment`).json())
}

// --- Workspace-level ---

export async function listWorkspaceHosts(wsId: string, params?: ListParams): Promise<HostList> {
  return apiRequest(
    infraApi.get(`workspaces/${wsId}/hosts`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function getWorkspaceHost(wsId: string, hostId: string): Promise<Host> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/hosts/${hostId}`).json())
}

export async function createWorkspaceHost(wsId: string, data: Pick<Host, "metadata" | "spec">): Promise<Host> {
  return apiRequest(infraApi.post(`workspaces/${wsId}/hosts`, { json: data }).json())
}

export async function updateWorkspaceHost(
  wsId: string,
  hostId: string,
  data: Pick<Host, "metadata" | "spec">,
): Promise<Host> {
  return apiRequest(infraApi.put(`workspaces/${wsId}/hosts/${hostId}`, { json: data }).json())
}

export async function deleteWorkspaceHost(wsId: string, hostId: string): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/hosts/${hostId}`).json())
}

export async function deleteWorkspaceHosts(wsId: string, ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/hosts`, { json: { ids } }).json())
}

export async function bindWorkspaceHostEnvironment(
  wsId: string,
  hostId: string,
  body: BindEnvironmentRequest,
): Promise<void> {
  await apiRequest(infraApi.post(`workspaces/${wsId}/hosts/${hostId}/bind-environment`, { json: body }).json())
}

export async function unbindWorkspaceHostEnvironment(wsId: string, hostId: string): Promise<void> {
  await apiRequest(infraApi.post(`workspaces/${wsId}/hosts/${hostId}/unbind-environment`).json())
}

// --- Namespace-level ---

export async function listNamespaceHosts(wsId: string, nsId: string, params?: ListParams): Promise<HostList> {
  return apiRequest(
    infraApi
      .get(`workspaces/${wsId}/namespaces/${nsId}/hosts`, { searchParams: params as Record<string, string> })
      .json(),
  )
}

export async function getNamespaceHost(wsId: string, nsId: string, hostId: string): Promise<Host> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/namespaces/${nsId}/hosts/${hostId}`).json())
}

export async function createNamespaceHost(
  wsId: string,
  nsId: string,
  data: Pick<Host, "metadata" | "spec">,
): Promise<Host> {
  return apiRequest(infraApi.post(`workspaces/${wsId}/namespaces/${nsId}/hosts`, { json: data }).json())
}

export async function updateNamespaceHost(
  wsId: string,
  nsId: string,
  hostId: string,
  data: Pick<Host, "metadata" | "spec">,
): Promise<Host> {
  return apiRequest(infraApi.put(`workspaces/${wsId}/namespaces/${nsId}/hosts/${hostId}`, { json: data }).json())
}

export async function deleteNamespaceHost(wsId: string, nsId: string, hostId: string): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/namespaces/${nsId}/hosts/${hostId}`).json())
}

export async function deleteNamespaceHosts(wsId: string, nsId: string, ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete(`workspaces/${wsId}/namespaces/${nsId}/hosts`, { json: { ids } }).json())
}

export async function bindNamespaceHostEnvironment(
  wsId: string,
  nsId: string,
  hostId: string,
  body: BindEnvironmentRequest,
): Promise<void> {
  await apiRequest(
    infraApi.post(`workspaces/${wsId}/namespaces/${nsId}/hosts/${hostId}/bind-environment`, { json: body }).json(),
  )
}

export async function unbindNamespaceHostEnvironment(wsId: string, nsId: string, hostId: string): Promise<void> {
  await apiRequest(
    infraApi.post(`workspaces/${wsId}/namespaces/${nsId}/hosts/${hostId}/unbind-environment`).json(),
  )
}
