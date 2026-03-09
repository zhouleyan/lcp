import { api, apiRequest } from "./client"
import { getAccessToken } from "@/lib/auth"
import type { User, UserList, ListParams, ChangePasswordRequest, StatusResponse, OIDCUserInfo, WorkspaceList, NamespaceList } from "./types"

export async function listUsers(params?: ListParams): Promise<UserList> {
  return apiRequest(api.get("users", { searchParams: params as Record<string, string> }).json())
}

export async function getUser(id: string): Promise<User> {
  return apiRequest(api.get(`users/${id}`).json())
}

export async function createUser(data: Pick<User, "metadata" | "spec">): Promise<User> {
  return apiRequest(api.post("users", { json: data }).json())
}

export async function updateUser(
  id: string,
  data: Pick<User, "metadata" | "spec">,
): Promise<User> {
  return apiRequest(api.put(`users/${id}`, { json: data }).json())
}

export async function deleteUser(id: string): Promise<void> {
  await apiRequest(api.delete(`users/${id}`).json())
}

export async function deleteUsers(ids: string[]): Promise<void> {
  await apiRequest(api.delete("users", { json: { ids } }).json())
}

export async function getWorkspaceUser(workspaceId: string, userId: string): Promise<User> {
  return apiRequest(api.get(`workspaces/${workspaceId}/users/${userId}`).json())
}

export async function getNamespaceUser(workspaceId: string, namespaceId: string, userId: string): Promise<User> {
  return apiRequest(api.get(`workspaces/${workspaceId}/namespaces/${namespaceId}/users/${userId}`).json())
}

export async function listWorkspaceUsers(
  workspaceId: string,
  params?: ListParams,
): Promise<UserList> {
  return apiRequest(
    api.get(`workspaces/${workspaceId}/users`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function addWorkspaceUsers(workspaceId: string, ids: string[]): Promise<void> {
  await apiRequest(api.post(`workspaces/${workspaceId}/users`, { json: { ids } }).json())
}

export async function removeWorkspaceUsers(workspaceId: string, ids: string[]): Promise<void> {
  await apiRequest(api.delete(`workspaces/${workspaceId}/users`, { json: { ids } }).json())
}

export async function listNamespaceUsers(
  workspaceId: string,
  namespaceId: string,
  params?: ListParams,
): Promise<UserList> {
  return apiRequest(
    api.get(`workspaces/${workspaceId}/namespaces/${namespaceId}/users`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function addNamespaceUsers(workspaceId: string, namespaceId: string, ids: string[]): Promise<void> {
  await apiRequest(api.post(`workspaces/${workspaceId}/namespaces/${namespaceId}/users`, { json: { ids } }).json())
}

export async function removeNamespaceUsers(workspaceId: string, namespaceId: string, ids: string[]): Promise<void> {
  await apiRequest(api.delete(`workspaces/${workspaceId}/namespaces/${namespaceId}/users`, { json: { ids } }).json())
}

export async function listUserWorkspaces(
  userId: string,
  params?: ListParams,
): Promise<WorkspaceList> {
  return apiRequest(
    api.get(`users/${userId}:workspaces`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function listUserNamespaces(
  userId: string,
  params?: ListParams,
): Promise<NamespaceList> {
  return apiRequest(
    api.get(`users/${userId}:namespaces`, { searchParams: params as Record<string, string> }).json(),
  )
}

export async function changePassword(
  userId: string,
  data: ChangePasswordRequest,
): Promise<StatusResponse> {
  return apiRequest(api.post(`users/${userId}/change-password`, { json: data }).json())
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
