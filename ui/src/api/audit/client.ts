import { api } from "../client"

export const auditApi = api.extend({ prefixUrl: "/api/audit/v1" })
