import { create } from "zustand"
import type { OIDCUserInfo } from "@/api/types"
import { getUserInfo } from "@/api/users"

interface AuthState {
  user: OIDCUserInfo | null
  loading: boolean
  fetchUser: () => Promise<void>
  clearUser: () => void
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  loading: false,
  fetchUser: async () => {
    set({ loading: true })
    try {
      const user = await getUserInfo()
      set({ user, loading: false })
    } catch {
      set({ user: null, loading: false })
    }
  },
  clearUser: () => set({ user: null }),
}))
