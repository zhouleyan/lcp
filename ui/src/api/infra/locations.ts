import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Location, LocationList, ListParams } from "../types"

export async function listLocations(params?: ListParams): Promise<LocationList> {
  return apiRequest(infraApi.get("locations", { searchParams: params as Record<string, string> }).json())
}

export async function getLocation(id: string): Promise<Location> {
  return apiRequest(infraApi.get(`locations/${id}`).json())
}

export async function createLocation(data: Pick<Location, "metadata" | "spec">): Promise<Location> {
  return apiRequest(infraApi.post("locations", { json: data }).json())
}

export async function updateLocation(id: string, data: Pick<Location, "metadata" | "spec">): Promise<Location> {
  return apiRequest(infraApi.put(`locations/${id}`, { json: data }).json())
}

export async function patchLocation(
  id: string,
  data: Partial<Pick<Location, "metadata" | "spec">>,
): Promise<Location> {
  return apiRequest(infraApi.patch(`locations/${id}`, { json: data }).json())
}

export async function deleteLocation(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`locations/${id}`).json())
}

export async function deleteLocations(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("locations", { json: { ids } }).json())
}
