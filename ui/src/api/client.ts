import ky, { HTTPError } from "ky"
import { toast } from "sonner"
import { getAccessToken, refreshAccessToken } from "@/lib/auth"
import { useScopeStore } from "@/stores/scope-store"
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

/** Throttle scope invalidation to avoid 403 → invalidate → re-fetch → 403 loops. */
let lastScopeInvalidateAt = 0
const SCOPE_INVALIDATE_COOLDOWN_MS = 10_000

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
        if (response.status === 403) {
          const now = Date.now()
          if (now - lastScopeInvalidateAt > SCOPE_INVALIDATE_COOLDOWN_MS) {
            lastScopeInvalidateAt = now
            useScopeStore.getState().invalidate()
          }
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
        // Use HTTP status code (numeric) instead of body's status field ("Failure" string)
        throw new ApiError({ ...apiBody, status: err.response.status })
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
  "must be 3-50 lowercase alphanumeric characters or hyphens": "api.validation.name.format",
  "must be >= 0": "api.validation.rackCapacity.min",
  "invalid IP address format": "api.validation.ip.format",
  "must be 3-50 lowercase alphanumeric characters or hyphens, starting and ending with alphanumeric": "api.validation.name.networkFormat",
}

const detailMessagePrefixMap: Record<string, string> = {
  "invalid CIDR format": "api.validation.cidr.format",
  "gateway ": "api.validation.gateway.notInRange",
  "CIDR ": "api.validation.cidr.overlap",
  "subnet CIDR ": "api.validation.cidr.notWithinNetwork",
  "must be at most ": "api.validation.description.tooLong",
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
  "cannot delete location": "api.error.cannotDeleteLocation",
  "cannot delete network": "api.error.cannotDeleteNetwork",
  "cannot delete subnet": "api.error.cannotDeleteSubnet",
  "network has reached the maximum number of subnets": "api.error.maxSubnetsReached",
  "subnet already has gateway": "api.error.subnetAlreadyHasGateway",
  "IP ": "api.error.ipNotInRange",
  "cannot delete CA": "api.error.cannotDeleteCA",
}

const reasonMessageMap: Record<string, string> = {
  Conflict: "api.error.conflict",
  NotFound: "api.error.notFound",
  BadRequest: "api.error.badRequest",
  Forbidden: "api.error.forbidden",
}

export function translateDetailMessage(message: string): string {
  if (detailMessageMap[message]) return detailMessageMap[message]
  for (const [prefix, key] of Object.entries(detailMessagePrefixMap)) {
    if (message.startsWith(prefix)) return key
  }
  return message
}

export function translateApiError(err: ApiError): string {
  if (messageMap[err.message]) return messageMap[err.message]
  for (const [prefix, key] of Object.entries(messagePrefixMap)) {
    if (err.message.startsWith(prefix)) return key
  }
  return reasonMessageMap[err.reason] ?? err.message
}

/**
 * Show a toast error for a caught exception. Handles both ApiError and unknown errors.
 * @param err - The caught error
 * @param t - The i18n translation function
 * @param resourceKey - Optional i18n key for the resource name (e.g. "user.title")
 */
export function showApiError(err: unknown, t: (key: string, params?: Record<string, string | number>) => string, resourceKey?: string) {
  if (err instanceof ApiError) {
    const i18nKey = translateApiError(err)
    const params = resourceKey ? { resource: t(resourceKey) } : undefined
    toast.error(i18nKey !== err.message ? t(i18nKey, params) : err.message)
  } else {
    toast.error(t("api.error.internalError"))
  }
}

/**
 * Handle API errors in form submissions by mapping backend errors to form field errors.
 * @param err - The caught error
 * @param form - react-hook-form's form instance (must have setError)
 * @param t - i18n translation function
 * @param i18nPrefix - The i18n key prefix for field names (e.g., "region", "site", "location")
 * @param resourceKey - The i18n key for the resource title (e.g., "region.title")
 */
export function handleFormApiError(
  err: unknown,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: { setError: (name: any, error: { message: string }) => void },
  t: (key: string, params?: Record<string, string | number>) => string,
  i18nPrefix: string,
  resourceKey: string,
) {
  if (err instanceof ApiError && err.details?.length) {
    for (const d of err.details) {
      const field = d.field.replace(/^(metadata|spec)\./, "")
      const i18nKey = translateDetailMessage(d.message)
      form.setError(field, {
        message: i18nKey !== d.message
          ? t(i18nKey, { field: t(`${i18nPrefix}.${field}`) || field })
          : d.message,
      })
    }
  } else if (err instanceof ApiError) {
    const i18nKey = translateApiError(err)
    form.setError("root", {
      message: i18nKey !== err.message
        ? t(i18nKey, { resource: t(resourceKey) })
        : err.message,
    })
  } else {
    form.setError("root", { message: t("api.error.internalError") })
  }
}

/** Default page size for select/dropdown data fetches (e.g., loading all regions for a select). */
export const SELECT_PAGE_SIZE = 200
