import { api } from "./client"
import { getAccessToken } from "@/lib/auth"
import type { User, UserList, ListParams, ChangePasswordRequest, StatusResponse, OIDCUserInfo } from "./types"

export async function listUsers(params?: ListParams): Promise<UserList> {
  return api.get("users", { searchParams: params as Record<string, string> }).json()
}

export async function getUser(id: string): Promise<User> {
  return api.get(`users/${id}`).json()
}

export async function createUser(data: Pick<User, "metadata" | "spec">): Promise<User> {
  return api.post("users", { json: data }).json()
}

export async function updateUser(
  id: string,
  data: Pick<User, "metadata" | "spec">,
): Promise<User> {
  return api.put(`users/${id}`, { json: data }).json()
}

export async function deleteUser(id: string): Promise<void> {
  await api.delete(`users/${id}`)
}

export async function deleteUsers(ids: string[]): Promise<void> {
  await api.delete("users", { json: { ids } })
}

export async function listWorkspaceUsers(
  workspaceId: string,
  params?: ListParams,
): Promise<UserList> {
  return api
    .get(`workspaces/${workspaceId}/users`, { searchParams: params as Record<string, string> })
    .json()
}

export async function addWorkspaceUsers(workspaceId: string, ids: string[]): Promise<void> {
  await api.post(`workspaces/${workspaceId}/users`, { json: { ids } })
}

export async function removeWorkspaceUsers(workspaceId: string, ids: string[]): Promise<void> {
  await api.delete(`workspaces/${workspaceId}/users`, { json: { ids } })
}

export async function listNamespaceUsers(
  namespaceId: string,
  params?: ListParams,
): Promise<UserList> {
  return api
    .get(`namespaces/${namespaceId}/users`, { searchParams: params as Record<string, string> })
    .json()
}

export async function addNamespaceUsers(namespaceId: string, ids: string[]): Promise<void> {
  await api.post(`namespaces/${namespaceId}/users`, { json: { ids } })
}

export async function removeNamespaceUsers(namespaceId: string, ids: string[]): Promise<void> {
  await api.delete(`namespaces/${namespaceId}/users`, { json: { ids } })
}

export async function changePassword(
  userId: string,
  data: ChangePasswordRequest,
): Promise<StatusResponse> {
  return api.post(`users/${userId}/change-password`, { json: data }).json()
}

export async function getUserInfo(): Promise<OIDCUserInfo> {
  const res = await fetch("/oidc/userinfo", {
    headers: { Authorization: `Bearer ${getAccessToken()}` },
  })
  if (!res.ok) {
    throw new Error("Failed to fetch user info")
  }
  return res.json()
}
