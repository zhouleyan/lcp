import { create } from "zustand"
import { persist } from "zustand/middleware"

interface ScopeState {
  workspaceId: string | null
  namespaceId: string | null
  /** Monotonically increasing counter; bump to force scope-selector to re-fetch lists. */
  version: number
  setWorkspace: (id: string | null) => void
  setNamespace: (id: string | null) => void
  setScope: (wsId: string | null, nsId: string | null) => void
  /** Call after creating, editing, or deleting a workspace or namespace. */
  invalidate: () => void
}

export const useScopeStore = create<ScopeState>()(
  persist(
    (set) => ({
      workspaceId: null,
      namespaceId: null,
      version: 0,
      setWorkspace: (id) => set({ workspaceId: id, namespaceId: null }),
      setNamespace: (id) => set({ namespaceId: id }),
      setScope: (wsId, nsId) => set({ workspaceId: wsId, namespaceId: nsId }),
      invalidate: () => set((s) => ({ version: s.version + 1 })),
    }),
    { name: "lcp-scope", partialize: (s) => ({ workspaceId: s.workspaceId, namespaceId: s.namespaceId }) },
  ),
)
