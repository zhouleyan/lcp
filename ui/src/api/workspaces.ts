import { api } from "./client"
import type { Workspace, WorkspaceList, ListParams } from "./types"

export async function listWorkspaces(params?: ListParams): Promise<WorkspaceList> {
  return api.get("workspaces", { searchParams: params as Record<string, string> }).json()
}

export async function getWorkspace(id: string): Promise<Workspace> {
  return api.get(`workspaces/${id}`).json()
}

export async function createWorkspace(
  data: Pick<Workspace, "metadata" | "spec">,
): Promise<Workspace> {
  return api.post("workspaces", { json: data }).json()
}

export async function updateWorkspace(
  id: string,
  data: Pick<Workspace, "metadata" | "spec">,
): Promise<Workspace> {
  return api.put(`workspaces/${id}`, { json: data }).json()
}

export async function deleteWorkspace(id: string): Promise<void> {
  await api.delete(`workspaces/${id}`)
}

export async function deleteWorkspaces(ids: string[]): Promise<void> {
  await api.delete("workspaces", { json: { ids } })
}
