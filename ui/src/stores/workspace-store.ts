import { create } from "zustand"
import { persist } from "zustand/middleware"

interface WorkspaceState {
  currentWorkspaceId: string | null
  currentNamespaceId: string | null
  setCurrentWorkspace: (id: string | null) => void
  setCurrentNamespace: (id: string | null) => void
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      currentWorkspaceId: null,
      currentNamespaceId: null,
      setCurrentWorkspace: (id) => set({ currentWorkspaceId: id, currentNamespaceId: null }),
      setCurrentNamespace: (id) => set({ currentNamespaceId: id }),
    }),
    { name: "lcp-workspace" },
  ),
)
