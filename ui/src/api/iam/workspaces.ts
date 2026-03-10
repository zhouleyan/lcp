import { iamApi } from "./client"
import { apiRequest } from "../client"
import type { Workspace, WorkspaceList, ListParams } from "../types"

export async function listWorkspaces(params?: ListParams): Promise<WorkspaceList> {
  return apiRequest(iamApi.get("workspaces", { searchParams: params as Record<string, string> }).json())
}

export async function getWorkspace(id: string): Promise<Workspace> {
  return apiRequest(iamApi.get(`workspaces/${id}`).json())
}

export async function createWorkspace(
  data: Pick<Workspace, "metadata" | "spec">,
): Promise<Workspace> {
  return apiRequest(iamApi.post("workspaces", { json: data }).json())
}

export async function updateWorkspace(
  id: string,
  data: Pick<Workspace, "metadata" | "spec">,
): Promise<Workspace> {
  return apiRequest(iamApi.put(`workspaces/${id}`, { json: data }).json())
}

export async function deleteWorkspace(id: string): Promise<void> {
  await apiRequest(iamApi.delete(`workspaces/${id}`).json())
}

export async function deleteWorkspaces(ids: string[]): Promise<void> {
  await apiRequest(iamApi.delete("workspaces", { json: { ids } }).json())
}
