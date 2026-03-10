import { create } from "zustand"
import type { UserPermissionsSpec } from "@/api/types"
import { getUserPermissions } from "@/api/iam/rbac"

interface PermissionState {
  permissions: UserPermissionsSpec | null
  loading: boolean
  fetchPermissions: (userId: string) => Promise<void>
  clearPermissions: () => void
}

let fetchPromise: Promise<void> | null = null

export const usePermissionStore = create<PermissionState>()((set) => ({
  permissions: null,
  loading: false,
  fetchPermissions: async (userId: string) => {
    if (fetchPromise) return fetchPromise
    fetchPromise = (async () => {
      set({ loading: true })
      try {
        const data = await getUserPermissions(userId)
        set({ permissions: data.spec, loading: false })
      } catch {
        set({ permissions: null, loading: false })
      } finally {
        fetchPromise = null
      }
    })()
    return fetchPromise
  },
  clearPermissions: () => set({ permissions: null }),
}))
