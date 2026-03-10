import { auditApi } from "./client"
import { apiRequest } from "../client"
import type { AuditLogList, AuditLog, ListParams } from "../types"

export async function listAuditLogs(params?: ListParams): Promise<AuditLogList> {
  return apiRequest(auditApi.get("logs", { searchParams: params as Record<string, string> }).json())
}

export async function getAuditLog(id: string): Promise<AuditLog> {
  return apiRequest(auditApi.get(`logs/${id}`).json())
}
