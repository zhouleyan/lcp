import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { Subnet, SubnetList, ListParams } from "../types"

export async function listSubnets(networkId: string, params?: ListParams): Promise<SubnetList> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets`, { searchParams: params as Record<string, string> }).json())
}

export async function getSubnet(networkId: string, subnetId: string): Promise<Subnet> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets/${subnetId}`).json())
}

export async function createSubnet(networkId: string, data: Pick<Subnet, "metadata" | "spec">): Promise<Subnet> {
  return apiRequest(networkApi.post(`networks/${networkId}/subnets`, { json: data }).json())
}

export async function updateSubnet(networkId: string, subnetId: string, data: Pick<Subnet, "metadata" | "spec">): Promise<Subnet> {
  return apiRequest(networkApi.put(`networks/${networkId}/subnets/${subnetId}`, { json: data }).json())
}

export async function deleteSubnet(networkId: string, subnetId: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets/${subnetId}`).json())
}

export async function deleteSubnets(networkId: string, ids: string[]): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets`, { json: { ids } }).json())
}
