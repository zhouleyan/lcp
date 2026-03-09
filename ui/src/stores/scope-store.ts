import { create } from "zustand"
import { persist } from "zustand/middleware"

interface ScopeState {
  workspaceId: string | null
  namespaceId: string | null
  setWorkspace: (id: string | null) => void
  setNamespace: (id: string | null) => void
}

export const useScopeStore = create<ScopeState>()(
  persist(
    (set) => ({
      workspaceId: null,
      namespaceId: null,
      setWorkspace: (id) => set({ workspaceId: id, namespaceId: null }),
      setNamespace: (id) => set({ namespaceId: id }),
    }),
    { name: "lcp-scope" },
  ),
)
