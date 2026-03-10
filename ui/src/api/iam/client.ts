import { api } from "../client"

export const iamApi = api.extend({ prefixUrl: "/api/iam/v1" })
