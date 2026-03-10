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

// Extend HTTPError to carry parsed API error body
interface HTTPErrorWithBody extends HTTPError {
  _apiBody?: StatusResponse
}

export const api = ky.create({
  prefixUrl: "/api",
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
      async (error: HTTPErrorWithBody) => {
        const { response } = error
        // For 4xx, parse body and attach to error so apiRequest can use it
        if (response.status >= 400 && response.status < 500) {
          try {
            const body: StatusResponse = await response.json()
            if (body.reason) {
              error._apiBody = body
            }
          } catch {
            // body not JSON or already consumed
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
        if (response.status >= 500) {
          window.location.href = `/error?status=${response.status}`
        }
        return response
      },
    ],
  },
})

/**
 * Wraps a ky request promise. Catches HTTPError (4xx) and converts to ApiError.
 */
export async function apiRequest<T>(request: Promise<T>): Promise<T> {
  try {
    return await request
  } catch (err) {
    if (err instanceof HTTPError) {
      const apiBody = (err as HTTPErrorWithBody)._apiBody
      if (apiBody) {
        throw new ApiError(apiBody)
      }
      throw new ApiError({
        apiVersion: "",
        kind: "Status",
        status: err.response.status,
        reason: err.response.statusText,
        message: err.response.statusText,
      })
    }
    throw err
  }
}

// Map backend English error messages to i18n keys for frontend translation.
const detailMessageMap: Record<string, string> = {
  "is required": "api.validation.required",
  "must be 3-50 alphanumeric characters or underscores": "api.validation.username.format",
  "is not a valid email address": "api.validation.email.format",
  "must be a valid Chinese mobile number (e.g. 13800138000)": "api.validation.phone.format",
  "must be 8-128 characters": "api.validation.password.length",
  "must contain at least one uppercase letter": "api.validation.password.uppercase",
  "must contain at least one lowercase letter": "api.validation.password.lowercase",
  "must contain at least one digit": "api.validation.password.digit",
  "must be 'active' or 'inactive'": "api.validation.status.format",
}

const messageMap: Record<string, string> = {
  "old password is incorrect": "api.error.oldPasswordIncorrect",
  "oldPassword and newPassword are required": "api.error.badRequest",
  "cannot remove workspace owner": "api.error.cannotRemoveOwner",
  "cannot remove namespace owner": "api.error.cannotRemoveOwner",
}

const messagePrefixMap: Record<string, string> = {
  "namespace member limit exceeded": "api.error.memberLimitExceeded",
  "cannot delete workspace": "api.error.cannotDeleteWorkspace",
  "cannot delete namespace": "api.error.cannotDeleteNamespace",
}

const reasonMessageMap: Record<string, string> = {
  Conflict: "api.error.conflict",
  NotFound: "api.error.notFound",
  BadRequest: "api.error.badRequest",
}

export function translateDetailMessage(message: string): string {
  return detailMessageMap[message] ?? message
}

export function translateApiError(err: ApiError): string {
  if (messageMap[err.message]) return messageMap[err.message]
  for (const [prefix, key] of Object.entries(messagePrefixMap)) {
    if (err.message.startsWith(prefix)) return key
  }
  return reasonMessageMap[err.reason] ?? err.message
}
