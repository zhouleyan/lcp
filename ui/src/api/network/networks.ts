import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { Network, NetworkList, ListParams } from "../types"

export async function listNetworks(params?: ListParams): Promise<NetworkList> {
  return apiRequest(networkApi.get("networks", { searchParams: params as Record<string, string> }).json())
}

export async function getNetwork(id: string): Promise<Network> {
  return apiRequest(networkApi.get(`networks/${id}`).json())
}

export async function createNetwork(data: Pick<Network, "metadata" | "spec">): Promise<Network> {
  return apiRequest(networkApi.post("networks", { json: data }).json())
}

export async function updateNetwork(id: string, data: Pick<Network, "metadata" | "spec">): Promise<Network> {
  return apiRequest(networkApi.put(`networks/${id}`, { json: data }).json())
}

export async function deleteNetwork(id: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${id}`).json())
}

export async function deleteNetworks(ids: string[]): Promise<void> {
  await apiRequest(networkApi.delete("networks", { json: { ids } }).json())
}
