import ky from "ky"
import { getAccessToken, refreshAccessToken, logout } from "@/lib/auth"

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
          logout()
        }
        return response
      },
    ],
  },
})
