import { infraApi } from "./client"
import { apiRequest } from "../client"
import type { Site, SiteList, LocationList, ListParams } from "../types"

export async function listSites(params?: ListParams): Promise<SiteList> {
  return apiRequest(infraApi.get("sites", { searchParams: params as Record<string, string> }).json())
}

export async function getSite(id: string): Promise<Site> {
  return apiRequest(infraApi.get(`sites/${id}`).json())
}

export async function createSite(data: Pick<Site, "metadata" | "spec">): Promise<Site> {
  return apiRequest(infraApi.post("sites", { json: data }).json())
}

export async function updateSite(id: string, data: Pick<Site, "metadata" | "spec">): Promise<Site> {
  return apiRequest(infraApi.put(`sites/${id}`, { json: data }).json())
}

export async function patchSite(id: string, data: Partial<Pick<Site, "metadata" | "spec">>): Promise<Site> {
  return apiRequest(infraApi.patch(`sites/${id}`, { json: data }).json())
}

export async function deleteSite(id: string): Promise<void> {
  await apiRequest(infraApi.delete(`sites/${id}`).json())
}

export async function deleteSites(ids: string[]): Promise<void> {
  await apiRequest(infraApi.delete("sites", { json: { ids } }).json())
}

export async function getSiteLocations(id: string, params?: ListParams): Promise<LocationList> {
  return apiRequest(infraApi.get(`sites/${id}/locations`, { searchParams: params as Record<string, string> }).json())
}
