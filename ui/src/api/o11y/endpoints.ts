import { o11yApi } from "./client"
import { apiRequest } from "../client"
import type { Endpoint, EndpointList, ListParams, ProbeResult } from "../types"

export async function listEndpoints(params?: ListParams): Promise<EndpointList> {
  return apiRequest(o11yApi.get("endpoints", { searchParams: params as Record<string, string> }).json())
}

export async function getEndpoint(id: string): Promise<Endpoint> {
  return apiRequest(o11yApi.get(`endpoints/${id}`).json())
}

export async function createEndpoint(data: Pick<Endpoint, "metadata" | "spec">): Promise<Endpoint> {
  return apiRequest(o11yApi.post("endpoints", { json: data }).json())
}

export async function updateEndpoint(id: string, data: Pick<Endpoint, "metadata" | "spec">): Promise<Endpoint> {
  return apiRequest(o11yApi.put(`endpoints/${id}`, { json: data }).json())
}

export async function patchEndpoint(id: string, data: Partial<Pick<Endpoint, "metadata" | "spec">>): Promise<Endpoint> {
  return apiRequest(o11yApi.patch(`endpoints/${id}`, { json: data }).json())
}

export async function deleteEndpoint(id: string): Promise<void> {
  await apiRequest(o11yApi.delete(`endpoints/${id}`).json())
}

export async function deleteEndpoints(ids: string[]): Promise<void> {
  await apiRequest(o11yApi.delete("endpoints", { json: { ids } }).json())
}

export async function probeEndpoint(id: string): Promise<ProbeResult> {
  return apiRequest(o11yApi.post(`endpoints/${id}/probe`).json())
}
