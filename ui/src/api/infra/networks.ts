import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { AvailableNetworkList } from "../types"

// --- Platform-level ---

export async function listInfraNetworks(): Promise<AvailableNetworkList> {
  return apiRequest(infraApi.get("networks").json())
}

// --- Workspace-level ---

export async function listWorkspaceInfraNetworks(wsId: string): Promise<AvailableNetworkList> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/networks`).json())
}

// --- Namespace-level ---

export async function listNamespaceInfraNetworks(
  wsId: string,
  nsId: string,
): Promise<AvailableNetworkList> {
  return apiRequest(infraApi.get(`workspaces/${wsId}/namespaces/${nsId}/networks`).json())
}
