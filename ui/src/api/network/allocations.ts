import { networkApi } from "./client"
import { apiRequest } from "../client"
import type { IPAllocation, IPAllocationList, ListParams } from "../types"

export async function listAllocations(networkId: string, subnetId: string, params?: ListParams): Promise<IPAllocationList> {
  return apiRequest(networkApi.get(`networks/${networkId}/subnets/${subnetId}/allocations`, { searchParams: params as Record<string, string> }).json())
}

export async function createAllocation(networkId: string, subnetId: string, data: Pick<IPAllocation, "spec">): Promise<IPAllocation> {
  return apiRequest(networkApi.post(`networks/${networkId}/subnets/${subnetId}/allocations`, { json: data }).json())
}

export async function deleteAllocation(networkId: string, subnetId: string, allocationId: string): Promise<void> {
  await apiRequest(networkApi.delete(`networks/${networkId}/subnets/${subnetId}/allocations/${allocationId}`).json())
}
