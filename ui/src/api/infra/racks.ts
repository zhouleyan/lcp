import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Rack, RackList, ListParams } from "../types"

export async function listRacks(params?: ListParams): Promise<RackList> {
  return apiRequest(infraApi.get("racks", { searchParams: params as Record<string, string> }).json())
}

export async function getRack(id: string): Promise<Rack> {
  return apiRequest(infraApi.get(`racks/${id}`).json())
}

export async function createRack(data: Pick<Rack, "metadata" | "spec">): Promise<Rack> {
  return apiRequest(infraApi.post("racks", { json: data }).json())
}

export async function updateRack(id: string, data: Pick<Rack, "metadata" | "spec">): Promise<Rack> {
  return apiRequest(infraApi.put(`racks/${id}`, { json: data }).json())
}

export async function patchRack(
  id: string,
  data: Partial<Pick<Rack, "metadata" | "spec">>,
): Promise<Rack> {
  return apiRequest(infraApi.patch(`racks/${id}`, { json: data }).json())
}

export async function deleteRack(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`racks/${id}`).json())
}

export async function deleteRacks(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("racks", { json: { ids } }).json())
}
