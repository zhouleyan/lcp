import { apiRequest } from "../client"
import type { Overview } from "../types"
import { dashboardApi } from "./client"

export async function getPlatformOverview(): Promise<Overview> {
  return apiRequest(dashboardApi.get("overview").json())
}

export async function getWorkspaceOverview(workspaceId: string): Promise<Overview> {
  return apiRequest(dashboardApi.get(`workspaces/${workspaceId}/overview`).json())
}

export async function getNamespaceOverview(workspaceId: string, namespaceId: string): Promise<Overview> {
  return apiRequest(dashboardApi.get(`workspaces/${workspaceId}/namespaces/${namespaceId}/overview`).json())
}
