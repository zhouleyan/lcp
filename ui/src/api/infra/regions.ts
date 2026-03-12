import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Region, RegionList, SiteList, ListParams } from "../types"

export async function listRegions(params?: ListParams): Promise<RegionList> {
  return apiRequest(infraApi.get("regions", { searchParams: params as Record<string, string> }).json())
}

export async function getRegion(id: string): Promise<Region> {
  return apiRequest(infraApi.get(`regions/${id}`).json())
}

export async function createRegion(data: Pick<Region, "metadata" | "spec">): Promise<Region> {
  return apiRequest(infraApi.post("regions", { json: data }).json())
}

export async function updateRegion(id: string, data: Pick<Region, "metadata" | "spec">): Promise<Region> {
  return apiRequest(infraApi.put(`regions/${id}`, { json: data }).json())
}

export async function patchRegion(id: string, data: Partial<Pick<Region, "metadata" | "spec">>): Promise<Region> {
  return apiRequest(infraApi.patch(`regions/${id}`, { json: data }).json())
}

export async function deleteRegion(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`regions/${id}`).json())
}

export async function deleteRegions(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("regions", { json: { ids } }).json())
}

export async function getRegionSites(id: string, params?: ListParams): Promise<SiteList> {
  return apiRequest(infraApi.get(`regions/${id}/sites`, { searchParams: params as Record<string, string> }).json())
}
