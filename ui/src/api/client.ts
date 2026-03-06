import ky, { HTTPError } from "ky"
import { getAccessToken, refreshAccessToken } from "@/lib/auth"
import type { StatusResponse, StatusResponseDetail } from "./types"

export class ApiError extends Error {
  status: number
  reason: string
  details?: StatusResponseDetail[]

  constructor(response: StatusResponse) {
    super(response.message)
    this.name = "ApiError"
    this.status = typeof response.status === "number" ? response.status : parseInt(String(response.status), 10)
    this.reason = response.reason
    this.details = response.details
  }
}

async function parseApiError(error: unknown): Promise<ApiError> {
  if (error instanceof HTTPError) {
    try {
      const body: StatusResponse = await error.response.json()
      if (body.reason) {
        return new ApiError(body)
      }
    } catch {
      // response body is not a valid StatusResponse
    }
    return new ApiError({
      apiVersion: "",
      kind: "Status",
      status: error.response.status,
      reason: error.response.statusText,
      message: error.message,
    })
  }
  if (error instanceof Error) {
    return new ApiError({
      apiVersion: "",
      kind: "Status",
      status: 0,
      reason: "Unknown",
      message: error.message,
    })
  }
  return new ApiError({
    apiVersion: "",
    kind: "Status",
    status: 0,
    reason: "Unknown",
    message: String(error),
  })
}

export const api = ky.create({
  prefixUrl: "/api/v1",
  hooks: {
    beforeRequest: [
      (request) => {
        const token = getAccessToken()
        if (token) {
          request.headers.set("Authorization", `Bearer ${token}`)
        }
      },
    ],
    beforeError: [
      async (error) => {
        const { response } = error
        // For 400/404/409, convert to ApiError so callers can handle structured errors
        if (response.status === 400 || response.status === 404 || response.status === 409) {
          try {
            const body: StatusResponse = await response.json()
            if (body.reason) {
              throw new ApiError(body)
            }
          } catch (e) {
            if (e instanceof ApiError) throw e
          }
        }
        return error
      },
    ],
    afterResponse: [
      async (request, _options, response) => {
        if (response.status === 401) {
          const refreshed = await refreshAccessToken()
          if (refreshed) {
            const token = getAccessToken()
            if (token) {
              request.headers.set("Authorization", `Bearer ${token}`)
            }
            return ky(request)
          }
          window.location.href = "/error?status=401"
          return response
        }
        if (response.status === 403 || response.status >= 500) {
          window.location.href = `/error?status=${response.status}`
        }
        return response
      },
    ],
  },
})

export { parseApiError }
