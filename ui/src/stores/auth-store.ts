import { create } from "zustand"
import type { OIDCUserInfo } from "@/api/types"
import { getUserInfo } from "@/api/iam/users"

interface AuthState {
  user: OIDCUserInfo | null
  loading: boolean
  fetchUser: () => Promise<void>
  clearUser: () => void
}

let fetchPromise: Promise<void> | null = null

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  loading: false,
  fetchUser: async () => {
    if (fetchPromise) return fetchPromise
    fetchPromise = (async () => {
      set({ loading: true })
      try {
        const user = await getUserInfo()
        set({ user, loading: false })
      } catch {
        set({ user: null, loading: false })
      } finally {
        fetchPromise = null
      }
    })()
    return fetchPromise
  },
  clearUser: () => set({ user: null }),
}))
